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

func TestTruncateSummary(t *testing.T) {
	tests := []struct {
		input    string
		model    db.Model
		expected string
	}{
		{"Short text", db.ModelFull, "Short text"},
		{"Text with model that cannot truncate", db.ModelOpenAI, "Text with model that cannot truncate"},
	}

	for _, tt := range tests {
		got := truncateSummary(tt.input, tt.model)
		if got != tt.expected {
			t.Errorf("truncateSummary(%q, %v) = %q, want %q", tt.input, tt.model, got, tt.expected)
		}
	}

	// Test truncation
	longText := ""
	for i := 0; i < 500; i++ {
		longText += "a"
	}
	got := truncateSummary(longText, db.ModelFull)
	if len(got) != 404 { // 400 'a's + " ..."
		t.Errorf("Expected length 404, got %d", len(got))
	}
	if got[400:] != " ..." {
		t.Errorf("Expected suffix ' ...', got %q", got[400:])
	}

	// Test multi-byte runes
	multiByte := ""
	for i := 0; i < 500; i++ {
		multiByte += "本" // 3 bytes
	}
	gotMulti := truncateSummary(multiByte, db.ModelFull)
	// 400 '本' runes * 3 bytes = 1200 bytes
	if len(gotMulti) != 1204 { // 1200 + 4 for " ..."
		t.Errorf("Expected length 1204, got %d", len(gotMulti))
	}
}

func BenchmarkEscapeXML(b *testing.B) {
	input := "Hello <world> & 'peace' \"joy\""
	for i := 0; i < b.N; i++ {
		escapeXML(input)
	}
}

func BenchmarkTruncateSummary(b *testing.B) {
	input := ""
	for i := 0; i < 500; i++ {
		input += "本" // 3 bytes per rune
	}
	m := db.ModelFull
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		truncateSummary(input, m)
	}
}
