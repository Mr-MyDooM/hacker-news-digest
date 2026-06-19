package template

import (
	"testing"
	"github.com/mj/hacker-news-digest/db"
)

func TestTruncateSummary(t *testing.T) {
	ts := globalFuncMap["truncateSummary"].(func(string, db.Model) string)
	tests := []struct {
		input    string
		model    db.Model
		expected string
	}{
		{"Short string", db.ModelPrefix, "Short string"},
		{string(make([]rune, 400)), db.ModelPrefix, string(make([]rune, 400))},
		{string(make([]rune, 401)), db.ModelPrefix, string(make([]rune, 400)) + " ..."},
		{"Long string that should be truncated but model is OpenAI", db.ModelOpenAI, "Long string that should be truncated but model is OpenAI"},
		{"Hello, 世界!", db.ModelPrefix, "Hello, 世界!"},
	}

	for _, tt := range tests {
		got := ts(tt.input, tt.model)
		if got != tt.expected {
			t.Errorf("truncateSummary(%q, %v) = %q, want %q", tt.input, tt.model, got, tt.expected)
		}
	}
}

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
