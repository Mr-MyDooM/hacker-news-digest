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
	truncateFunc := globalFuncMap["truncateSummary"].(func(string, db.Model) string)

	tests := []struct {
		name     string
		input    string
		model    db.Model
		expected string
	}{
		{
			name:     "Short string, can truncate",
			input:    "Hello",
			model:    db.ModelFull,
			expected: "Hello",
		},
		{
			name:     "Long string, can truncate",
			input:    strings.Repeat("a", 401),
			model:    db.ModelFull,
			expected: strings.Repeat("a", 400) + " ...",
		},
		{
			name:     "Long string, cannot truncate",
			input:    strings.Repeat("a", 401),
			model:    db.ModelOpenAI,
			expected: strings.Repeat("a", 401),
		},
		{
			name:     "Exactly 400 runes",
			input:    strings.Repeat("a", 400),
			model:    db.ModelFull,
			expected: strings.Repeat("a", 400),
		},
		{
			name:     "UTF-8 multi-byte characters",
			input:    strings.Repeat("世", 401),
			model:    db.ModelFull,
			expected: strings.Repeat("世", 400) + " ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateFunc(tt.input, tt.model)
			if got != tt.expected {
				t.Errorf("truncateSummary() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func BenchmarkEscapeXML(b *testing.B) {
	input := "Hello & <world> \"quoted\" 'entities'"
	for i := 0; i < b.N; i++ {
		escapeXML(input)
	}
}

func BenchmarkTruncateSummary(b *testing.B) {
	summary := strings.Repeat("This is a long summary that should be truncated eventually because it is way too long for a single line in the digest. ", 10)
	model := db.ModelFull
	truncateFunc := globalFuncMap["truncateSummary"].(func(string, db.Model) string)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncateFunc(summary, model)
	}
}
