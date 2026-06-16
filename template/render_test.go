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
			name:     "Short string, no truncation",
			input:    "Short summary",
			model:    db.ModelPrefix,
			expected: "Short summary",
		},
		{
			name:     "Long string, truncation allowed",
			input:    strings.Repeat("a", 401),
			model:    db.ModelPrefix,
			expected: strings.Repeat("a", 400) + " ...",
		},
		{
			name:     "Long string, truncation NOT allowed",
			input:    strings.Repeat("a", 401),
			model:    db.ModelOpenAI,
			expected: strings.Repeat("a", 401),
		},
		{
			name:     "Multi-byte characters",
			input:    strings.Repeat("世", 401),
			model:    db.ModelPrefix,
			expected: strings.Repeat("世", 400) + " ...",
		},
	}

	truncateSummary := globalFuncMap["truncateSummary"].(func(string, db.Model) string)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateSummary(tt.input, tt.model)
			if got != tt.expected {
				if len(got) > 50 {
					t.Errorf("truncateSummary() length = %d, want %d", len(got), len(tt.expected))
				} else {
					t.Errorf("truncateSummary() = %q, want %q", got, tt.expected)
				}
			}
		})
	}
}
