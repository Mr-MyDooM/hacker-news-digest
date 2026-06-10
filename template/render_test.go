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
	f := globalFuncMap["truncateSummary"].(func(string, db.Model) string)
	model := db.ModelFull

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Short string",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "Exactly 400 chars",
			input:    strings.Repeat("a", 400),
			expected: strings.Repeat("a", 400),
		},
		{
			name:     "401 chars",
			input:    strings.Repeat("a", 401),
			expected: strings.Repeat("a", 400) + " ...",
		},
		{
			name:     "Multi-byte characters",
			input:    strings.Repeat("世", 401),
			expected: strings.Repeat("世", 400) + " ...",
		},
		{
			name:     "No truncate model",
			input:    strings.Repeat("a", 500),
			expected: strings.Repeat("a", 500),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model
			if tt.name == "No truncate model" {
				m = db.ModelOpenAI
			}
			got := f(tt.input, m)
			if got != tt.expected {
				t.Errorf("truncateSummary() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func BenchmarkEscapeXML(b *testing.B) {
	s := `This is a "test" & it has <some> 'special' characters.`
	for i := 0; i < b.N; i++ {
		escapeXML(s)
	}
}

func BenchmarkTruncateSummary(b *testing.B) {
	s := strings.Repeat("This is a long summary with many characters to test the performance of the truncation logic. ", 10)
	model := db.ModelFull
	f := globalFuncMap["truncateSummary"].(func(string, db.Model) string)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f(s, model)
	}
}
