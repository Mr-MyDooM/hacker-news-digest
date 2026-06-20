package hn

import (
	"testing"
)

func TestIsBlockedContent(t *testing.T) {
	tests := []struct {
		content string
		blocked bool
	}{
		{"Normal content", false},
		{"Something went wrong", true},
		{"Log in to Twitter", true},
		{"JUST A MOMENT", true}, // case insensitive check
		{"ddos protection", true},
		{"regular news article about technology", false},
	}

	for _, tt := range tests {
		if got := isBlockedContent(tt.content); got != tt.blocked {
			t.Errorf("isBlockedContent(%q) = %v, want %v", tt.content, got, tt.blocked)
		}
	}
}
