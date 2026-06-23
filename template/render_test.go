package template

import (
	"html/template"
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

func TestHelpers(t *testing.T) {
	cleanTitle := globalFuncMap["cleanTitle"].(func(string) string)
	if got := cleanTitle("Show HN: Test"); got != "Test" {
		t.Errorf("cleanTitle failed, got %q", got)
	}

	titleBadge := globalFuncMap["titleBadge"].(func(string) template.HTML)
	if got := string(titleBadge("Show HN: Test")); got == "" {
		t.Error("titleBadge failed to return badge")
	}
}
