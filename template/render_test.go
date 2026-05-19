package template

import (
	"strings"
	"testing"
	"time"

	"github.com/mj/hacker-news-digest/extractor"
	"github.com/mj/hacker-news-digest/hn"
)

func TestRenderFeed(t *testing.T) {
	newsList := []*hn.News{
		{
			Title:      "Test Title & More",
			URL:        "https://example.com/a?b=c&d=e",
			Score:      100,
			SubmitTime: time.Now(),
			Summary:    "Summary <script>alert(1)</script>",
			Image: &extractor.WebImage{
				URL: "https://example.com/img.jpg?x=y&z=w",
			},
			CommentURL: "https://news.ycombinator.com/item?id=123&foo=bar",
		},
	}
	siteURL := "https://hndigest.example.com?a=b&c=d"

	feed := RenderFeed(newsList, siteURL)

	// Check header siteURL escaping
	if !strings.Contains(feed, "<id>https://hndigest.example.com?a=b&amp;c=d/</id>") {
		t.Errorf("siteURL not escaped in <id>")
	}
	if !strings.Contains(feed, "href=\"https://hndigest.example.com?a=b&amp;c=d/feed.xml\"") {
		t.Errorf("siteURL not escaped in <link href>")
	}
	if !strings.Contains(feed, "<uri>https://hndigest.example.com?a=b&amp;c=d</uri>") {
		t.Errorf("siteURL not escaped in <author><uri>")
	}

	// Check news entry escaping
	if !strings.Contains(feed, "<title>Test Title &amp; More</title>") {
		t.Errorf("news title not escaped")
	}
	if !strings.Contains(feed, "src=\"https://example.com/img.jpg?x=y&amp;z=w\"") {
		t.Errorf("image URL not escaped")
	}
	if !strings.Contains(feed, "href=\"https://hndigest.example.com?a=b&amp;c=d/#") {
		t.Errorf("siteURL not escaped in summary link")
	}
	if !strings.Contains(feed, "href=\"https://news.ycombinator.com/item?id=123&amp;foo=bar\"") {
		t.Errorf("comment URL not escaped")
	}
	if !strings.Contains(feed, "Summary &lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Errorf("summary not escaped")
	}
}
