package hn

import (
	"testing"
)

func BenchmarkIsBlockedContent(b *testing.B) {
	content := "This is some content that is definitely not blocked but it is quite long so it takes some time to lowercase it all. Let's make it even longer to see the impact of lowercasing the whole thing every time."
	for i := 0; i < b.N; i++ {
		isBlockedContent(content)
	}
}
