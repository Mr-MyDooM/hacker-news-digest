package hn

import (
	"testing"
)

func TestIsBlockedContent(t *testing.T) {
	tests := []struct {
		content  string
		expected bool
	}{
		{"This is normal content", false},
		{"Something went wrong with the page", true},
		{"Please enable JavaScript to see this content", true},
		{"Access denied to this resource", true},
		{"Just a moment... checking your browser", true},
		{"Follow us on Twitter", false},
	}

	for _, tt := range tests {
		got := isBlockedContent(tt.content)
		if got != tt.expected {
			t.Errorf("isBlockedContent(%q) = %v, want %v", tt.content, got, tt.expected)
		}
	}
}

func BenchmarkIsBlockedContent(b *testing.B) {
	content := "This is a relatively long piece of content that does not contain any blocked patterns. " +
		"It is intended to simulate a typical article body where we check for blocks. " +
		"The quick brown fox jumps over the lazy dog. " +
		"Go is an open source programming language that makes it easy to build simple, reliable, and efficient software."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isBlockedContent(content)
	}
}
