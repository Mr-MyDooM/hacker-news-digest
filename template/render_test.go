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
		name     string
		input    string
		model    db.Model
		expected string
	}{
		{
			name:     "no truncation needed",
			input:    "hello world",
			model:    db.ModelPrefix,
			expected: "hello world",
		},
		{
			name:     "model cannot truncate",
			input:    strings.Repeat("a", 500),
			model:    db.ModelOpenAI,
			expected: strings.Repeat("a", 500),
		},
		{
			name:     "truncation needed",
			input:    strings.Repeat("a", 500),
			model:    db.ModelPrefix,
			expected: strings.Repeat("a", 400) + " ...",
		},
		{
			name:     "multi-byte characters",
			input:    "世界" + strings.Repeat("a", 399),
			model:    db.ModelPrefix,
			expected: "世界" + strings.Repeat("a", 398) + " ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateFn(tt.input, tt.model)
			if got != tt.expected {
				t.Errorf("truncateSummary() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func BenchmarkEscapeXML(b *testing.B) {
	s := "This is a <test> & it has \"quotes\" and 'apostrophes'."
	for i := 0; i < b.N; i++ {
		escapeXML(s)
	}
}

func BenchmarkTruncateSummary(b *testing.B) {
	s := strings.Repeat("This is a long summary that should be truncated at some point. ", 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globalFuncMap["truncateSummary"].(func(string, db.Model) string)(s, db.ModelPrefix)
	}
}
