package template

import (
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

func BenchmarkEscapeXML(b *testing.B) {
	input := "Hello & welcome <to> the 'world' of \"Go\"! <script>alert('xss')</script>"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		escapeXML(input)
	}
}

func BenchmarkTruncateSummary(b *testing.B) {
	input := "Go is an open source programming language that makes it easy to build simple, reliable, and efficient software. " +
		"It is a compiled, statically typed language designed at Google. " +
		"Go is syntactically similar to C, but with memory safety, garbage collection, structural typing, and CSP-style concurrency. " +
		"The language is often referred to as Golang because of its domain name, golang.org, but its proper name is Go. " +
		"世界こんにちは！ 🚀 This is some multi-byte text to ensure rune counting is involved. " +
		"More text to ensure we exceed the 400 rune limit easily. " +
		"Repeat: " +
		"Go is an open source programming language that makes it easy to build simple, reliable, and efficient software. " +
		"It is a compiled, statically typed language designed at Google. " +
		"Go is syntactically similar to C, but with memory safety, garbage collection, structural typing, and CSP-style concurrency. " +
		"The language is often referred to as Golang because of its domain name, golang.org, but its proper name is Go."

	truncateSummary := globalFuncMap["truncateSummary"].(func(string, db.Model) string)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncateSummary(input, db.ModelFull)
	}
}
