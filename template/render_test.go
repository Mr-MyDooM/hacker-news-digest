package template

import (
	"github.com/mj/hacker-news-digest/db"
	"strings"
	"testing"
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
		{strings.Repeat("a", 400), db.ModelFull, strings.Repeat("a", 400)},
		{strings.Repeat("a", 401), db.ModelFull, strings.Repeat("a", 400) + " ..."},
		{strings.Repeat("a", 401), db.ModelOpenAI, strings.Repeat("a", 401)}, // OpenAI doesn't truncate
		{"こんにちは", db.ModelFull, "こんにちは"},
		{strings.Repeat("あ", 401), db.ModelFull, strings.Repeat("あ", 400) + " ..."},
	}

	for _, tt := range tests {
		got := truncateFn(tt.input, tt.model)
		if got != tt.expected {
			t.Errorf("truncateSummary(%d chars, %s) = %d chars, want %d chars", len([]rune(tt.input)), tt.model, len([]rune(got)), len([]rune(tt.expected)))
		}
	}
}

func BenchmarkEscapeXML(b *testing.B) {
	input := "Hello & welcome to <Hacker News>! It's \"great\" to see you."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		escapeXML(input)
	}
}

func BenchmarkTruncateSummary(b *testing.B) {
	input := strings.Repeat("This is a long summary that needs to be truncated. ", 20)
	model := db.ModelFull
	truncateFn := globalFuncMap["truncateSummary"].(func(string, db.Model) string)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncateFn(input, model)
	}
}
