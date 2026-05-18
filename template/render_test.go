package template

import (
	"strings"
	"testing"
	"time"

	"github.com/mj/hacker-news-digest/hn"
)

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello & World", "Hello &amp; World"},
		{"<script>", "&lt;script&gt;"},
		{"\"Quotes\"", "&quot;Quotes&quot;"},
		{"'Single'", "&apos;Single&apos;"},
		{"Mixed < & > \" '", "Mixed &lt; &amp; &gt; &quot; &apos;"},
	}

	for _, tt := range tests {
		if got := escapeXML(tt.input); got != tt.expected {
			t.Errorf("escapeXML(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestRenderFeedEscaping(t *testing.T) {
	newsList := []*hn.News{
		{
			Title:      "Malicious <script>alert(1)</script>",
			URL:        "https://example.com/?a=b&c=d",
			Score:      100,
			Summary:    "Dangerous Summary <img src=x onerror=alert(1)>",
			SubmitTime: time.Now(),
			Author:     "Attacker's Name",
		},
	}

	siteURL := "https://site.com/\"'><script>"
	feed := RenderFeed(newsList, siteURL)

	if strings.Contains(feed, "<script>") {
		t.Errorf("Feed contains unescaped <script> tag")
	}
	if strings.Contains(feed, "& ") {
		t.Errorf("Feed contains unescaped & character")
	}

	// Check if title is escaped
	expectedTitle := "&lt;script&gt;alert(1)&lt;/script&gt;"
	if !strings.Contains(feed, expectedTitle) {
		t.Errorf("Feed title not correctly escaped")
	}

	// Check if siteURL in ID and Link is escaped
	escapedSiteURL := "https://site.com/&quot;&apos;&gt;&lt;script&gt;"
	if !strings.Contains(feed, escapedSiteURL) {
		t.Errorf("siteURL not correctly escaped in feed")
	}
}
