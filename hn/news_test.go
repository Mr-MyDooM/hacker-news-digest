package hn

import (
	"testing"
)

func TestIsBlockedContent(t *testing.T) {
	tests := []struct {
		content  string
		expected bool
	}{
		{"Normal content", false},
		{"Something went wrong here", true},
		{"JUST A MOMENT", true},
		{"Please enable JavaScript to continue", true},
		{"This is a great article about Go", false},
	}

	for _, tt := range tests {
		if got := isBlockedContent(tt.content); got != tt.expected {
			t.Errorf("isBlockedContent(%q) = %v, want %v", tt.content, got, tt.expected)
		}
	}
}
