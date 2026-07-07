package extractor

import (
	"fmt"
	"io"
	"log"
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
	Images      []string
	SiteName    string
}

func Extract(url string, maxLength int) (*ExtractResult, error) {
	client := GetSafeClient(30 * time.Second)
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

	// Collect all image candidates: body images first (more relevant), then meta
	bodyImgs := collectBodyImages(doc, url)
	result.Images = append(result.Images, bodyImgs...)
	if result.Image != "" {
		result.Images = append(result.Images, result.Image)
	}

	// Pick the first available image candidate
	if result.Image == "" && len(bodyImgs) > 0 {
		result.Image = bodyImgs[0]
	}

	// Validate content quality: if text looks like noise (low alphabetic ratio),
	// try Jina as an alternative source before giving up.
	if !isValidContent(result.Content) {
		log.Printf("Poor content quality for %s (alpha ratio %.2f), retrying via Jina", url, alphaRatio(result.Content))
		if jinaResult, err := ExtractViaJina(url, maxLength); err == nil {
			jinaResult.Image = result.Image
			return jinaResult, nil
		}
	}

	// When body content is empty (paywalled, login wall, etc.), try Jina reader as proxy.
	if result.Content == "" {
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

func collectBodyImages(doc *goquery.Document, pageURL string) []string {
	skipKeywords := []string{"avatar", "spinner", "icon", "logo", "button", "banner", "thumb", "sprite", "loading", "placeholder", "pixel"}
	seen := make(map[string]bool)
	var candidates []string

	addIfValid := func(src string) {
		if src == "" || strings.HasPrefix(src, "data:") || seen[src] {
			return
		}
		seen[src] = true
		candidates = append(candidates, resolveURL(pageURL, src))
	}

	// Preferred: images inside article/main containers
	containers := doc.Find("article, main, [role=main], .post-content, .entry-content, .article-body, [class*=content], [id*=content]")
	containers.Find("img").Each(func(i int, sel *goquery.Selection) {
		src, _ := sel.Attr("src")
		cls, _ := sel.Attr("class")
		id, _ := sel.Attr("id")
		alt, _ := sel.Attr("alt")
		attrStr := strings.ToLower(cls + " " + id + " " + alt)
		for _, kw := range skipKeywords {
			if strings.Contains(attrStr, kw) {
				return
			}
		}
		addIfValid(src)
	})

	// Fallback: all document images (skip small ones by HTML attrs)
	doc.Find("img").Each(func(i int, sel *goquery.Selection) {
		src, _ := sel.Attr("src")
		if seen[resolveURL(pageURL, src)] {
			return
		}
		w, wex := sel.Attr("width")
		h, hex := sel.Attr("height")
		if wex && hex && isDimSmall(w, h) {
			return
		}
		cls, _ := sel.Attr("class")
		id, _ := sel.Attr("id")
		alt, _ := sel.Attr("alt")
		attrStr := strings.ToLower(cls + " " + id + " " + alt)
		for _, kw := range skipKeywords {
			if strings.Contains(attrStr, kw) {
				return
			}
		}
		addIfValid(src)
	})

	return candidates
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
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	if strings.HasPrefix(href, "/") {
		parts := strings.SplitN(base, "/", 4)
		if len(parts) >= 3 {
			return parts[0] + "//" + parts[2] + href
		}
	}
	base = strings.TrimRight(base, "/")
	return base + "/" + strings.TrimLeft(href, "/")
}

func extractContent(doc *goquery.Document, maxLength int) string {
	// Remove non-content elements aggressively
	doc.Find("script, style, nav, footer, header, iframe, form, noscript, svg, canvas, aside").Remove()
	doc.Find("[class*=sidebar], [class*=comment], [class*=widget], [class*=meta], [class*=menu], [class*=nav-], [class*=footer], [id*=sidebar], [id*=comment], [id*=footer]").Remove()
	doc.Find("[class*=related], [id*=related], [class*=recommend], [id*=recommend], [class*=suggestion], [id*=suggestion]").Remove()
	doc.Find("[class*=discussion], [id*=discussion], [class*=thread], [id*=thread]").Remove()

	article := doc.Find("article").First()
	if article.Length() > 0 {
		return cleanText(article.Text(), maxLength)
	}

	main := doc.Find("main, [role=main], #content, .content, .post-content, .entry-content, .article-body, #main-content, .post-body, .article-content, .story-body").First()
	if main.Length() > 0 {
		return cleanText(main.Text(), maxLength)
	}

	body := doc.Find("body")
	if body.Length() == 0 {
		return ""
	}

	// Score all child divs/sections recursively by text length, link density, and pattern matching
	best := body
	bestScore := 0.0
	findBestContent(body, titleText(doc), &best, &bestScore)

	if bestScore > 0 {
		return cleanText(best.Text(), maxLength)
	}

	return cleanText(body.Text(), maxLength)
}

func titleText(doc *goquery.Document) string {
	return strings.ToLower(strings.TrimSpace(doc.Find("title").First().Text()))
}

// contentBoost returns a multiplier for elements with positive content class/id patterns.
// Copy of Python's impact_factor logic (Readability-inspired).
func contentBoost(child *goquery.Selection) float64 {
	cls, _ := child.Attr("class")
	id, _ := child.Attr("id")
	attr := strings.ToLower(cls + " " + id)

	positivePatterns := []string{"article", "content", "post", "main", "entry", "story", "text", "body", "page"}
	negativePatterns := []string{"sidebar", "comment", "footer", "header", "nav", "menu", "widget", "ad-", "advertisement", "promo", "sponsor", "meta", "related", "recommend"}

	for _, p := range positivePatterns {
		if strings.Contains(attr, p) {
			return 2.0
		}
	}
	for _, p := range negativePatterns {
		if strings.Contains(attr, p) {
			return 0.2
		}
	}
	return 1.0
}

func findBestContent(sel *goquery.Selection, title string, best **goquery.Selection, bestScore *float64) {
	sel.Children().Each(func(i int, child *goquery.Selection) {
		tag := goquery.NodeName(child)
		if tag != "div" && tag != "section" && tag != "article" && tag != "main" {
			findBestContent(child, title, best, bestScore)
			return
		}
		text := strings.TrimSpace(child.Text())
		textLen := utf8.RuneCountInString(text)
		ld := linkDensity(child)
		if textLen < 50 || ld >= 0.5 {
			findBestContent(child, title, best, bestScore)
			return
		}
		boost := contentBoost(child)

		// LCS title match boost: Python gives high scores to headers matching the page title
		titleBoost := 1.0
		if tag == "h1" || tag == "h2" || tag == "h3" || tag == "h4" {
			childText := strings.ToLower(text)
			if lcsRatio(title, childText) > 0.85 {
				titleBoost = 3.0
			}
		}

		// Alphabetic ratio boost: prefer prose, penalize only when very noisy
		alphaBoost := alphaRatio(text)
		if alphaBoost < 0.3 {
			alphaBoost = 0.2 // heavily penalize non-prose (code, gibberish)
		} else if alphaBoost < 0.5 {
			alphaBoost = 0.7 // mildly penalize mixed content
		}

		score := float64(textLen) * boost * titleBoost * alphaBoost / (ld + 0.1)
		if score > *bestScore {
			*bestScore = score
			*best = child
		}
		findBestContent(child, title, best, bestScore)
	})
}

// lcsRatio computes the ratio of the longest common subsequence length to the longer string length.
func lcsRatio(a, b string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	m, n := len(a), len(b)
	// Use small table if both strings are short, otherwise fast approximation
	if m > 200 || n > 200 {
		// Approximate: count common words
		wordsA := strings.Fields(a)
		wordsB := strings.Fields(b)
		if len(wordsA) == 0 || len(wordsB) == 0 {
			return 0
		}
		setB := make(map[string]bool, len(wordsB))
		for _, w := range wordsB {
			setB[w] = true
		}
		common := 0
		for _, w := range wordsA {
			if setB[w] {
				common++
			}
		}
		return float64(common) / float64(max(len(wordsA), len(wordsB)))
	}
	// Full LCS for short strings using space-optimized O(min(M, N)) DP.
	if n > m {
		a, b = b, a
		m, n = n, m
	}
	// m >= n, so we use n for the row size.
	var prev, curr []int
	if n <= 200 {
		// Use stack-allocated buffers for small strings to avoid heap allocations.
		var buf [2][201]int
		prev = buf[0][:n+1]
		curr = buf[1][:n+1]
	} else {
		prev = make([]int, n+1)
		curr = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				curr[j] = prev[j-1] + 1
			} else {
				curr[j] = max(prev[j], curr[j-1])
			}
		}
		prev, curr = curr, prev
	}
	lcs := prev[n]
	return float64(lcs) / float64(m)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func linkDensity(sel *goquery.Selection) float64 {
	text := strings.TrimSpace(sel.Text())
	if len(text) == 0 {
		return 1
	}

	var linkTextBuilder strings.Builder
	sel.Find("a").Each(func(i int, a *goquery.Selection) {
		linkTextBuilder.WriteString(a.Text())
	})
	linkTextLen := len(strings.TrimSpace(linkTextBuilder.String()))

	return float64(linkTextLen) / float64(len(text))
}

// isValidContent checks that extracted text looks like real article content, not noise.
func isValidContent(text string) bool {
	if len(text) < 200 {
		return false
	}
	return alphaRatio(text) >= 0.50
}

// alphaRatio returns the fraction of non-space characters that are alphabetic.
func alphaRatio(text string) float64 {
	if len(text) == 0 {
		return 0
	}
	alpha := 0
	total := 0
	for _, r := range text {
		if r == ' ' || r == '\n' || r == '\t' || r == '\r' {
			continue
		}
		total++
		if unicode.IsLetter(r) {
			alpha++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(alpha) / float64(total)
}

// isGibberishLine detects lines that are unlikely to be real prose:
// too many non-alphanumeric characters, code fragments, or noise.
func isGibberishLine(line string) bool {
	alpha := 0
	special := 0
	codeChars := 0
	runeCount := 0
	for _, r := range line {
		runeCount++
		if unicode.IsLetter(r) {
			alpha++
		} else if !unicode.IsSpace(r) && !unicode.IsDigit(r) && !unicode.IsPunct(r) {
			special++
		}

		if r == '{' || r == '}' || r == '(' || r == ')' || r == ';' || r == '=' || r == '<' || r == '>' {
			codeChars++
		}
	}
	if runeCount < 5 {
		return true
	}
	// If >30% of chars are non-standard special chars, likely noise
	if special > 0 && float64(special)/float64(runeCount) > 0.30 {
		return true
	}
	// If <40% alphabetic, too many symbols/code
	if float64(alpha)/float64(runeCount) < 0.40 {
		return true
	}
	// Code-like lines: too many brackets/semicolons
	if codeChars > 0 && float64(codeChars)/float64(runeCount) > 0.10 {
		return true
	}
	return false
}

func cleanText(text string, maxLength int) string {
	// Performance: Optimized truncation using utf8.RuneCountInString and range loop
	// to avoid expensive []rune allocations. Reduces latency for long articles.
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "\r", "")

	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip very short lines (noise, navigation, metadata)
		wordCount := countWords(line)
		if wordCount < 10 {
			continue
		}
		// Skip gibberish and code-like lines
		if isGibberishLine(line) {
			continue
		}
		// Skip lines that look like metadata (dates, author, share, tags)
		skipMeta := []string{"published", "updated", "written by", "posted by", "by ", "share this",
			"tweet", "facebook", "linkedin", "tags:", "category:", "subscribe", "comments",
			"reply", "leave a", "©", "all rights reserved", "privacy", "cookie"}
		lower := strings.ToLower(line)
		isMeta := false
		for _, m := range skipMeta {
			if strings.HasPrefix(lower, m) {
				isMeta = true
				break
			}
		}
		if isMeta && wordCount < 20 {
			continue
		}
		cleaned = append(cleaned, line)
	}

	text = strings.Join(cleaned, "\n")
	text = collapseSpaces(text)

	if utf8.RuneCountInString(text) > maxLength {
		count := 0
		for i := range text {
			if count == maxLength {
				text = text[:i]
				break
			}
			count++
		}
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
	client := GetSafeClient(30 * time.Second)
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
