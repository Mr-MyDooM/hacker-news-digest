package extractor

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"
)

type PDFExtractor struct {
	data []byte
	url  string
}

func NewPDFExtractor(data []byte, url string) *PDFExtractor {
	return &PDFExtractor{data: data, url: url}
}

func (p *PDFExtractor) GetContent(maxLength int) string {
	r, err := pdf.NewReader(bytes.NewReader(p.data), int64(len(p.data)))
	if err != nil {
		return ""
	}

	var paragraphs []string
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		lines := strings.Split(text, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if len(line) < 3 {
				continue
			}

			if isTOCLine(line) {
				continue
			}

			if !isNormalParagraph(line) {
				continue
			}

			paragraphs = append(paragraphs, line)
		}
	}

	text := strings.Join(paragraphs, "\n")

	runes := []rune(text)
	if len(runes) > maxLength {
		text = string(runes[:maxLength])
		if lastSpace := strings.LastIndex(text, " "); lastSpace > 0 {
			text = text[:lastSpace] + " ..."
		}
	}

	return text
}

func (p *PDFExtractor) getFavicon() string {
	parts := strings.SplitN(p.url, "/", 3)
	if len(parts) >= 2 {
		return parts[0] + "//" + parts[1] + "/favicon.ico"
	}
	return "/favicon.ico"
}

func isTOCLine(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) < 3 {
		return true
	}

	dotCount := 0
	digitCount := 0
	for _, r := range line {
		if r == '.' {
			dotCount++
		}
		if r >= '0' && r <= '9' {
			digitCount++
		}
	}

	if dotCount > 5 && digitCount > 0 {
		return true
	}

	return false
}

func isNormalParagraph(line string) bool {
	words := strings.Fields(line)
	if len(words) < 3 {
		return false
	}

	asciiWords := 0
	for _, w := range words {
		allASCII := true
		for _, r := range w {
			if r > 127 {
				allASCII = false
				break
			}
		}
		if allASCII {
			asciiWords++
		}
	}

	dotCount := 0
	for _, r := range line {
		if r == '.' {
			dotCount++
		}
	}

	if float64(dotCount)/float64(utf8.RuneCountInString(line)) > 0.3 {
		return false
	}

	return true
}

func isPDF(data []byte) bool {
	return len(data) > 4 && string(data[:5]) == "%PDF-"
}

func pdfTitle(data []byte) string {
	text := string(data)
	idx := strings.Index(text, "/Title(")
	if idx < 0 {
		return ""
	}
	idx += 7
	end := strings.Index(text[idx:], ")")
	if end < 0 {
		return ""
	}
	title := text[idx : idx+end]
	title = strings.ReplaceAll(title, "\\", "")
	return title
}
