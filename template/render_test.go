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
	tests := []struct {
		name     string
		input    string
		model    db.Model
		expected string
	}{
		{
			name:     "Short string",
			input:    "Hello",
			model:    db.ModelFull,
			expected: "Hello",
		},
		{
			name:     "Long string",
			input:    strings.Repeat("a", 500),
			model:    db.ModelFull,
			expected: strings.Repeat("a", 400) + " ...",
		},
		{
			name:     "Multi-byte characters",
			input:    strings.Repeat("😊", 500),
			model:    db.ModelFull,
			expected: strings.Repeat("😊", 400) + " ...",
		},
		{
			name:     "Model that cannot truncate",
			input:    strings.Repeat("a", 500),
			model:    db.ModelOpenAI,
			expected: strings.Repeat("a", 500),
		},
	}

	truncateSummary := globalFuncMap["truncateSummary"].(func(string, db.Model) string)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateSummary(tt.input, tt.model)
			if got != tt.expected {
				t.Errorf("truncateSummary() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func BenchmarkTruncateSummary(b *testing.B) {
	s := strings.Repeat("This is a test string with some multi-byte characters like 😊. ", 100)
	m := db.ModelFull
	truncateSummary := globalFuncMap["truncateSummary"].(func(string, db.Model) string)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncateSummary(s, m)
	}
}
