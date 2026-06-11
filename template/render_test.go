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

func TestTruncateSummary(t *testing.T) {
	truncateFn := globalFuncMap["truncateSummary"].(func(string, db.Model) string)
	tests := []struct {
		input    string
		model    db.Model
		expected string
	}{
		{"short", db.ModelFull, "short"},
		{strings.Repeat("a", 401), db.ModelFull, strings.Repeat("a", 400) + " ..."},
		{strings.Repeat("a", 401), db.ModelOpenAI, strings.Repeat("a", 401)}, // OpenAI cannot truncate
	}

	for _, tt := range tests {
		got := truncateFn(tt.input, tt.model)
		if got != tt.expected {
			t.Errorf("truncateSummary(%d chars) = %d chars, want %d chars", len(tt.input), len(got), len(tt.expected))
		}
	}
}

func BenchmarkEscapeXML(b *testing.B) {
	s := `This is a "test" & it has <some> 'special' characters.`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		escapeXML(s)
	}
}

func BenchmarkTruncateSummary(b *testing.B) {
	s := strings.Repeat("a", 500)
	truncateFn := globalFuncMap["truncateSummary"].(func(string, db.Model) string)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncateFn(s, db.ModelFull)
	}
}
