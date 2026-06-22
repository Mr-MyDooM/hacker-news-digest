package hn

import (
	"strings"
	"testing"
)

func BenchmarkIsBlockedContent(b *testing.B) {
	content := `This is a long piece of content that might contain some blocked patterns.
	Maybe it says something went wrong or access denied.
	But mostly it is just normal text to make the benchmark realistic.
	` + strings.Repeat("more normal text ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isBlockedContent(content)
	}
}

func BenchmarkIsBlockedContentNoMatch(b *testing.B) {
	content := strings.Repeat("this is just normal text without any blocked patterns. ", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isBlockedContent(content)
	}
}

func TestIsBlockedContent(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{"Normal content", false},
		{"Something went wrong here", true},
		{"Access denied", true},
		{"PLEASE ENABLE JAVASCRIPT", true},
	}
	for _, tt := range tests {
		if got := isBlockedContent(tt.content); got != tt.want {
			t.Errorf("isBlockedContent(%q) = %v, want %v", tt.content, got, tt.want)
		}
	}
}
