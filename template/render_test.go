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
		{"Short summary", db.ModelPrefix, "Short summary"},
		{"Non-truncatable model", db.Model("gpt-4"), "Non-truncatable model"},
		{strings.Repeat("a", 400), db.ModelPrefix, strings.Repeat("a", 400)},
		{strings.Repeat("a", 401), db.ModelPrefix, strings.Repeat("a", 400) + " ..."},
		{strings.Repeat("世界", 200), db.ModelPrefix, strings.Repeat("世界", 200)},
		{strings.Repeat("世界", 201), db.ModelPrefix, strings.Repeat("世界", 200) + " ..."},
	}

	for _, tt := range tests {
		got := truncateFn(tt.input, tt.model)
		if got != tt.expected {
			t.Errorf("truncateSummary(%d chars, %s) = %d chars, want %d chars", len(tt.input), tt.model, len(got), len(tt.expected))
			if len(got) < 100 {
				t.Errorf("  got: %q", got)
				t.Errorf("  want: %q", tt.expected)
			}
		}
	}
}
