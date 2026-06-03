package template

import (
	"strings"
	"testing"

	"github.com/mj/hacker-news-digest/db"
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
	input := "This is a <test> with & multiple \"special\" 'characters' to escape. " + strings.Repeat("more text & more <tags>. ", 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		escapeXML(input)
	}
}

func BenchmarkTruncateSummary(b *testing.B) {
	longSummary := strings.Repeat("This is a long summary for benchmarking purpose. It contains many characters and should be truncated. ", 20)
	// Current implementation uses db.Model to decide if it should truncate
	// We'll use a model that CanTruncate() returns true.
	m := db.ModelFull

	// Get the function from the map
	truncateFn := globalFuncMap["truncateSummary"].(func(string, db.Model) string)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncateFn(longSummary, m)
	}
}
