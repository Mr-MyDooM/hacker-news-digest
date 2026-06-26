package template

import (
	"strings"
	"testing"
	"github.com/mj/hacker-news-digest/db"
)

func BenchmarkEscapeXML(b *testing.B) {
	input := `This is a "test" & it has <some> 'special' characters. ` + strings.Repeat("more text for length. ", 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		escapeXML(input)
	}
}

func BenchmarkTruncateSummary(b *testing.B) {
	input := strings.Repeat("This is a long summary that should be truncated eventually because it is longer than four hundred runes. ", 10)
	model := db.ModelFull
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		globalFuncMap["truncateSummary"].(func(string, db.Model) string)(input, model)
	}
}
