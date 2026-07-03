package template

import (
	"strings"
	"testing"
	"time"

	"github.com/mj/hacker-news-digest/extractor"
	"github.com/mj/hacker-news-digest/hn"
)

func TestRenderAccessibility(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	data := &PageData{
		NewsList: []*hn.News{
			{
				Title:  "Test Story",
				URL:    "https://example.com",
				Score:  100,
				Author: "testauthor",
				Image: &extractor.WebImage{
					URL: "https://example.com/image.jpg",
				},
				Summary: "This is a test summary.",
			},
		},
		LastUpdated: time.Now(),
		Lang:        "en",
		DailyLinks:  []string{"2025-01-01"},
	}

	html, err := Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Check for aria-haspopup="true"
	if !strings.Contains(html, `aria-haspopup="true"`) {
		t.Errorf("Expected html to contain aria-haspopup=\"true\"")
	}

	// Check for modal accessibility attributes
	expectedModalAttrs := []string{
		`role="dialog"`,
		`aria-modal="true"`,
		`aria-labelledby="modal-title"`,
		`aria-label="Close"`,
		`id="modal-title"`,
	}

	for _, attr := range expectedModalAttrs {
		if !strings.Contains(html, attr) {
			t.Errorf("Expected html to contain %s", attr)
		}
	}
}
