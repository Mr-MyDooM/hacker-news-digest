package template

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/mj/hacker-news-digest/config"
	"github.com/mj/hacker-news-digest/db"
	"github.com/mj/hacker-news-digest/hn"
)

//go:embed *.gohtml
var templateFS embed.FS

var (
	cachedTmpl *template.Template
	tmplOnce   sync.Once
	tmplErr    error
)

type PageData struct {
	NewsList           []*hn.News
	LastUpdated        time.Time
	Lang               string
	DailyLinks         []string
	Path               string
	Site               string
	AdsenseID          string
	DisableAds         bool
	DisableTranslation bool
}

// Performance: globalFuncMap is defined at package level to avoid redundant allocations on every Render call.
var globalFuncMap = template.FuncMap{
	"slug": func(n *hn.News) string {
		return n.Slug()
	},
	"truncateSummary": func(s string, m db.Model) string {
		if m.CanTruncate() && len([]rune(s)) > 400 {
			return string([]rune(s)[:400]) + " ..."
		}
		return s
	},
	"formatTime": func(t time.Time) string {
		return t.Format("2006-01-02 15:04:05 MST")
	},
	"timeAgo": func(t time.Time) string {
		d := time.Since(t)
		switch {
		case d < time.Minute:
			return "just now"
		case d < time.Hour:
			m := int(d.Minutes())
			if m == 1 {
				return "1 minute ago"
			}
			return fmt.Sprintf("%d minutes ago", m)
		case d < 24*time.Hour:
			h := int(d.Hours())
			if h == 1 {
				return "1 hour ago"
			}
			return fmt.Sprintf("%d hours ago", h)
		case d < 7*24*time.Hour:
			days := int(d.Hours() / 24)
			if days == 1 {
				return "1 day ago"
			}
			return fmt.Sprintf("%d days ago", days)
		case d < 30*24*time.Hour:
			weeks := int(d.Hours() / (24 * 7))
			if weeks == 1 {
				return "1 week ago"
			}
			return fmt.Sprintf("%d weeks ago", weeks)
		case d < 365*24*time.Hour:
			months := int(d.Hours() / (24 * 30))
			if months == 1 {
				return "1 month ago"
			}
			return fmt.Sprintf("%d months ago", months)
		default:
			years := int(d.Hours() / (24 * 365))
			if years == 1 {
				return "1 year ago"
			}
			return fmt.Sprintf("%d years ago", years)
		}
	},
	"domain": func(url string) string {
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimPrefix(url, "http://")
		parts := strings.SplitN(url, "/", 2)
		return strings.TrimPrefix(parts[0], "www.")
	},
	"hasPrefix": strings.HasPrefix,
	"join":      strings.Join,
	"titleBadge": func(title string) template.HTML {
		var cls, label string
		switch {
		case strings.HasPrefix(title, "Show HN:"):
			cls, label = "show-hn", "Show HN"
		case strings.HasPrefix(title, "Ask HN:"):
			cls, label = "ask-hn", "Ask HN"
		case strings.HasPrefix(title, "Launch HN:"):
			cls, label = "launch-hn", "Launch HN"
		}
		if cls == "" {
			return ""
		}
		return template.HTML(fmt.Sprintf(`<span class="story-badge badge-%s">%s</span>`, cls, label))
	},
	"mod":     func(a, b int) int { return a % b },
	"safeCSS": func(s string) template.CSS { return template.CSS(s) },

	"cleanTitle": func(title string) string {
		trimmed := title
		for _, prefix := range []string{"Show HN:", "Ask HN:", "Launch HN:"} {
			if strings.HasPrefix(trimmed, prefix) {
				return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
			}
		}
		return trimmed
	},
}

// Init pre-parses templates from the embedded filesystem and caches them.
// Performance: This avoids expensive filesystem I/O and parsing overhead on every page render.
func Init() error {
	tmplOnce.Do(func() {
		cachedTmpl, tmplErr = template.New("base.gohtml").Funcs(globalFuncMap).ParseFS(templateFS, "*.gohtml")
	})
	return tmplErr
}

// Render executes the cached templates with the provided PageData.
// Performance: It uses the pre-parsed cachedTmpl to significantly speed up rendering.
func Render(data *PageData) (string, error) {
	Init()

	if tmplErr != nil {
		return "", fmt.Errorf("parse templates: %w", tmplErr)
	}

	var buf strings.Builder
	if err := cachedTmpl.ExecuteTemplate(&buf, "base.gohtml", data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

func RenderFeed(newsList []*hn.News, siteURL string) string {
	escapedSiteURL := escapeXML(siteURL)
	now := time.Now().In(config.IST)
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<feed xmlns="http://www.w3.org/2005/Atom">` + "\n")
	b.WriteString(fmt.Sprintf("  <title>HN Summary</title>\n"))
	b.WriteString(fmt.Sprintf("  <updated>%s</updated>\n", now.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("  <id>%s/</id>\n", escapedSiteURL))
	b.WriteString(fmt.Sprintf("  <link href=\"%s/feed.xml\" rel=\"self\"/>\n", escapedSiteURL))
	b.WriteString(fmt.Sprintf("  <author><name>Hacker News Digest</name><uri>%s</uri></author>\n", escapedSiteURL))

	for _, news := range newsList {
		if news.Score <= 20 {
			continue
		}
		imgTag := ""
		if news.Image != nil {
			imgTag = fmt.Sprintf("<img src=\"%s\" style=\"%s\"/><br/>", escapeXML(news.Image.URL), escapeXML(news.Image.GetSizeStyle(220)))
		}
		summaryLink := fmt.Sprintf(" <a href=\"%s/#%s\">[summary]</a>", escapedSiteURL, escapeXML(news.Slug()))
		commentsLink := ""
		if news.CommentURL != "" {
			commentsLink = fmt.Sprintf(" <a href=\"%s\">[comments]</a>", escapeXML(news.CommentURL))
		}

		content := imgTag + escapeXML(news.Summary) + summaryLink + commentsLink

		b.WriteString(fmt.Sprintf("  <entry>\n"))
		b.WriteString(fmt.Sprintf("    <title>%s</title>\n", escapeXML(news.Title)))
		b.WriteString(fmt.Sprintf("    <link href=\"%s\"/>\n", escapeXML(news.URL)))
		b.WriteString(fmt.Sprintf("    <id>%s/#%s</id>\n", escapedSiteURL, escapeXML(news.Slug())))
		b.WriteString(fmt.Sprintf("    <updated>%s</updated>\n", news.SubmitTime.Format(time.RFC3339)))
		b.WriteString(fmt.Sprintf("    <content type=\"html\"><![CDATA[%s]]></content>\n", strings.ReplaceAll(content, "]]>", "]]&gt;")))
		if news.Author != "" {
			b.WriteString(fmt.Sprintf("    <author><name>%s</name></author>\n", escapeXML(news.Author)))
		}
		b.WriteString(fmt.Sprintf("  </entry>\n"))
	}

	b.WriteString("</feed>\n")
	return b.String()
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

type writeWrapper struct {
	w io.Writer
}

func (w writeWrapper) Write(p []byte) (int, error) {
	return w.w.Write(p)
}
