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
			"Short summary",
			"Short summary",
			db.ModelFull,
			"Short summary",
		},
		{
			"Long summary",
			strings.Repeat("a", 401),
			db.ModelFull,
			strings.Repeat("a", 400) + " ...",
		},
		{
			"Unicode summary",
			strings.Repeat("你", 401),
			db.ModelFull,
			strings.Repeat("你", 400) + " ...",
		},
		{
			"Non-truncatable model",
			strings.Repeat("a", 401),
			db.ModelOpenAI,
			strings.Repeat("a", 401),
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
