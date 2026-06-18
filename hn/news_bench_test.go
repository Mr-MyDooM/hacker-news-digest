package hn

import (
	"strings"
	"testing"
)

func BenchmarkIsBlockedContent(b *testing.B) {
	content := `This is a long piece of content that does not contain any blocked patterns.
	It's just some normal text that we might find in an article.
	We want to make sure that the isBlockedContent function is efficient even when it doesn't find a match.
	Hacker News is a great place to find technical articles.
	This content is definitely not blocked.
	Wait, maybe I should make it even longer to be more realistic.
	` + strings.Repeat(" repetitive text ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isBlockedContent(content)
	}
}
