package extractor

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
)

type ExtractResult struct {
	Title       string
	Content     string
	Description string
	Favicon     string
	Image       string
	SiteName    string
}

func Extract(url string, maxLength int) (*ExtractResult, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	// On non-200 (e.g. 403), still attempt meta extraction — many sites block scrapers
	// but leave og:description in the initial HTML response.
	softFail := resp.StatusCode != http.StatusOK

	// Security: limit response body to 10MB to prevent resource exhaustion
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if softFail && len(body) == 0 {
		return nil, fmt.Errorf("fetch %s: status %d", url, resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")

	if isPDF(body) || strings.Contains(contentType, "pdf") {
		pe := NewPDFExtractor(body, url)
		result := &ExtractResult{}
		result.Content = pe.GetContent(maxLength)
		result.Title = pdfTitle(body)
		result.Favicon = pe.getFavicon()
		return result, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	result := &ExtractResult{}
	result.Title = extractTitle(doc)
	result.Description = extractMeta(doc, "description")
	result.Favicon = extractFavicon(doc, url)
	result.Image = extractMetaImage(doc, url)
	result.SiteName = extractMeta(doc, "og:site_name")
	result.Content = extractContent(doc, maxLength)

	// Fallback: no og:image found, look for first suitable <img> in article body
	if result.Image == "" {
		result.Image = extractBodyImage(doc, url)
	}

	// When body content is empty (paywalled, login wall, etc.), try Jina reader as proxy.
	if result.Content == "" && result.Description == "" {
		if jinaResult, err := ExtractViaJina(url, maxLength); err == nil {
			jinaResult.Image = result.Image // preserve og:image from original
			return jinaResult, nil
		}
	}

	return result, nil
}

func extractTitle(doc *goquery.Document) string {
	title := doc.Find("title").First().Text()
	if title != "" {
		return strings.TrimSpace(title)
	}
	return extractMeta(doc, "og:title")
}

func extractMeta(doc *goquery.Document, name string) string {
	selectors := []string{
		fmt.Sprintf(`meta[name="%s"]`, name),
		fmt.Sprintf(`meta[property="%s"]`, name),
		fmt.Sprintf(`meta[property="og:%s"]`, name),
		fmt.Sprintf(`meta[name="twitter:%s"]`, name),
	}
	for _, sel := range selectors {
		content, exists := doc.Find(sel).Attr("content")
		if exists && content != "" {
			return strings.TrimSpace(content)
		}
	}
	return ""
}

func extractMetaImage(doc *goquery.Document, pageURL string) string {
	selectors := []string{
		`meta[property="og:image"]`,
		`meta[name="twitter:image"]`,
		`meta[property="og:image:secure_url"]`,
	}
	for _, sel := range selectors {
		content, exists := doc.Find(sel).Attr("content")
		if exists && content != "" {
			content = strings.TrimSpace(content)
			if strings.HasPrefix(content, "http://") || strings.HasPrefix(content, "https://") {
				return content
			}
			return resolveURL(pageURL, content)
		}
	}
	return ""
}

func extractBodyImage(doc *goquery.Document, pageURL string) string {
	skipKeywords := []string{"avatar", "spinner", "icon", "logo", "button", "banner", "thumb", "sprite", "loading", "placeholder", "pixel"}
	candidate := ""

	// Search image containers first (article, main)
	containers := doc.Find("article, main, [role=main], .post-content, .entry-content, .article-body")
	if containers.Length() > 0 {
		containers.Find("img").Each(func(i int, sel *goquery.Selection) {
			if candidate != "" {
				return
			}
			src, exists := sel.Attr("src")
			if !exists || src == "" || strings.HasPrefix(src, "data:") {
				return
			}
			cls, _ := sel.Attr("class")
			id, _ := sel.Attr("id")
			alt, _ := sel.Attr("alt")
			attrStr := cls + " " + id + " " + alt
			lower := strings.ToLower(attrStr)
			for _, kw := range skipKeywords {
				if strings.Contains(lower, kw) {
					return
				}
			}
			candidate = resolveURL(pageURL, src)
		})
	}

	// Fallback: search full document for first suitable img
	if candidate == "" {
		doc.Find("img").Each(func(i int, sel *goquery.Selection) {
			if candidate != "" {
				return
			}
			src, exists := sel.Attr("src")
			if !exists || src == "" || strings.HasPrefix(src, "data:") {
				return
			}
			w, wex := sel.Attr("width")
			h, hex := sel.Attr("height")
			if wex && hex {
				if isDimSmall(w, h) {
					return
				}
			}
			cls, _ := sel.Attr("class")
			id, _ := sel.Attr("id")
			alt, _ := sel.Attr("alt")
			attrStr := cls + " " + id + " " + alt
			lower := strings.ToLower(attrStr)
			for _, kw := range skipKeywords {
				if strings.Contains(lower, kw) {
					return
				}
			}
			candidate = resolveURL(pageURL, src)
		})
	}

	return candidate
}

func isDimSmall(w, h string) bool {
	wid, errW := parseInt(w)
	hei, errH := parseInt(h)
	if errW == nil && errH == nil && (wid < 80 || hei < 80) {
		return true
	}
	return false
}

func parseInt(s string) (int, error) {
	s = strings.TrimRight(s, "px")
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func extractFavicon(doc *goquery.Document, pageURL string) string {
	sel := doc.Find(`link[rel="icon"]`).First()
	if sel.Length() == 0 {
		sel = doc.Find(`link[rel="shortcut icon"]`).First()
	}
	href, exists := sel.Attr("href")
	if !exists || href == "" {
		return pageURL + "/favicon.ico"
	}
	if strings.HasPrefix(href, "http") {
		return href
	}
	return resolveURL(pageURL, href)
}

func resolveURL(base, href string) string {
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	if strings.HasPrefix(href, "/") {
		parts := strings.SplitN(base, "/", 4)
		if len(parts) >= 3 {
			return parts[0] + "//" + parts[2] + href
		}
	}
	return base + "/" + strings.TrimLeft(href, "/")
}

func extractContent(doc *goquery.Document, maxLength int) string {
	doc.Find("script, style, nav, footer, header, iframe, form, noscript, svg, canvas, aside").Remove()

	article := doc.Find("article").First()
	if article.Length() > 0 {
		return cleanText(article.Text(), maxLength)
	}

	main := doc.Find("main, [role=main], #content, .content, .post-content, .entry-content, .article-body").First()
	if main.Length() > 0 {
		return cleanText(main.Text(), maxLength)
	}

	body := doc.Find("body")
	if body.Length() == 0 {
		return ""
	}

	best := body
	bestLen := 0
	body.Children().Each(func(i int, sel *goquery.Selection) {
		tag := goquery.NodeName(sel)
		if tag == "div" || tag == "section" || tag == "article" {
			text := sel.Text()
			textLen := utf8.RuneCountInString(strings.TrimSpace(text))
			linkDensity := linkDensity(sel)
			if textLen > bestLen && linkDensity < 0.5 {
				bestLen = textLen
				best = sel
			}
		}
	})

	if bestLen > 0 {
		return cleanText(best.Text(), maxLength)
	}

	doc.Find("p, h1, h2, h3, h4, h5, h6, li").Each(func(i int, sel *goquery.Selection) {
		text := strings.TrimSpace(sel.Text())
		if utf8.RuneCountInString(text) > 20 {
			// append to result, we'll collect manually
		}
	})

	return cleanText(body.Text(), maxLength)
}

func linkDensity(sel *goquery.Selection) float64 {
	text := strings.TrimSpace(sel.Text())
	linkText := ""
	sel.Find("a").Each(func(i int, a *goquery.Selection) {
		linkText += a.Text()
	})
	linkText = strings.TrimSpace(linkText)
	if len(text) == 0 {
		return 1
	}
	return float64(len(linkText)) / float64(len(text))
}

func cleanText(text string, maxLength int) string {
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "\r", "")

	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		wordCount := countWords(line)
		if wordCount < 3 {
			continue
		}
		cleaned = append(cleaned, line)
	}

	text = strings.Join(cleaned, "\n")
	text = collapseSpaces(text)

	runes := []rune(text)
	if len(runes) > maxLength {
		text = string(runes[:maxLength])
		if lastSpace := strings.LastIndex(text, " "); lastSpace > 0 {
			text = text[:lastSpace] + " ..."
		}
	}

	return text
}

func collapseSpaces(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\t' {
			if !prevSpace {
				if r == '\n' {
					b.WriteRune('\n')
				} else {
					b.WriteRune(' ')
				}
				prevSpace = true
			}
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return b.String()
}

func countWords(s string) int {
	count := 0
	inWord := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			count++
			inWord = true
		}
	}
	return count
}

// ExtractViaJina uses the Jina reader proxy to get readable content from pages
// that block direct scraping (paywalls, login walls, 403s).
func ExtractViaJina(url string, maxLength int) (*ExtractResult, error) {
	jinaURL := "https://r.jina.ai/" + url
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", jinaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("jina request: %w", err)
	}
	req.Header.Set("Accept", "text/plain")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jina fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jina status %d", resp.StatusCode)
	}

	// Security: limit jina response to 5MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("jina read: %w", err)
	}

	content := cleanText(string(body), maxLength)
	if content == "" {
		return nil, fmt.Errorf("jina returned empty content")
	}
	return &ExtractResult{Content: content}, nil
}
