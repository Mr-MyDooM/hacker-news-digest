package template

import (
	"testing"
	"time"
	"github.com/mj/hacker-news-digest/hn"
)

func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"hello", "hello"},
		{"&", "&amp;"},
		{"<", "&lt;"},
		{">", "&gt;"},
		{"\"", "&quot;"},
		{"'", "&apos;"},
		{"<script>alert(1)</script>", "&lt;script&gt;alert(1)&lt;/script&gt;"},
		{"&lt;b&gt;", "&amp;lt;b&amp;gt;"},
		{"' OR 1=1", "&apos; OR 1=1"},
		{"\" onclick=\"alert(1)", "&quot; onclick=&quot;alert(1)"},
	}

	for _, tt := range tests {
		got := escapeXML(tt.input)
		if got != tt.expected {
			t.Errorf("escapeXML(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func BenchmarkEscapeXML(b *testing.B) {
	input := "<script>alert('Hello & World')</script> \"quoted\""
	for i := 0; i < b.N; i++ {
		escapeXML(input)
	}
}

func BenchmarkRender(b *testing.B) {
	data := &PageData{
		NewsList: []*hn.News{
			{Title: "Test Story", URL: "http://example.com", Score: 100, Author: "user", Summary: "Summary text"},
		},
		LastUpdated: time.Now(),
		Site: "http://localhost",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Render(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}
