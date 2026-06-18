package hn

import (
	"testing"
)

func TestIsBlockedContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "Normal content",
			content: "This is some normal content that should not be blocked.",
			want:    false,
		},
		{
			name:    "Blocked content - something went wrong",
			content: "Oops! Something went wrong here.",
			want:    true,
		},
		{
			name:    "Blocked content - javascript required",
			content: "Please enable JavaScript to view this page.",
			want:    true,
		},
		{
			name:    "Blocked content - access denied",
			content: "403 Forbidden: Access denied.",
			want:    true,
		},
		{
			name:    "Mixed case content",
			content: "SOMETHING WENT WRONG",
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBlockedContent(tt.content); got != tt.want {
				t.Errorf("isBlockedContent() = %v, want %v", got, tt.want)
			}
		})
	}
}
