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
	truncateSummary := globalFuncMap["truncateSummary"].(func(string, db.Model) string)
	tests := []struct {
		name     string
		input    string
		model    db.Model
		expected string
	}{
		{
			name:     "no truncation needed",
			input:    "short string",
			model:    db.ModelFull,
			expected: "short string",
		},
		{
			name:     "truncation not allowed for model",
			input:    strings.Repeat("a", 500),
			model:    db.ModelOpenAI,
			expected: strings.Repeat("a", 500),
		},
		{
			name:     "truncation needed",
			input:    strings.Repeat("a", 500),
			model:    db.ModelFull,
			expected: strings.Repeat("a", 400) + " ...",
		},
		{
			name:     "truncation with non-ASCII",
			input:    strings.Repeat("😀", 500),
			model:    db.ModelFull,
			expected: strings.Repeat("😀", 400) + " ...",
		},
		{
			name:     "exact length",
			input:    strings.Repeat("a", 400),
			model:    db.ModelFull,
			expected: strings.Repeat("a", 400),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateSummary(tt.input, tt.model)
			if got != tt.expected {
				t.Errorf("truncateSummary() = %d chars, want %d chars", len([]rune(got)), len([]rune(tt.expected)))
				if len(got) < 100 {
					t.Errorf("got %q, want %q", got, tt.expected)
				}
			}
		})
	}
}

func BenchmarkTruncateSummary(b *testing.B) {
	s := strings.Repeat("This is a fairly long string with some non-ASCII characters like 😀 to make it interesting. ", 20)
	m := db.ModelFull
	f := globalFuncMap["truncateSummary"].(func(string, db.Model) string)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f(s, m)
	}
}

func BenchmarkEscapeXML(b *testing.B) {
	s := "This is a <test> string with & multiple \"entities\" and 'single' quotes to be escaped."
	for i := 0; i < b.N; i++ {
		escapeXML(s)
	}
}
