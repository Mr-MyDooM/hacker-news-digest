package llm

import "github.com/mj/hacker-news-digest/db"

type Summarizer interface {
	Summarize(content string) (string, db.Model, error)
}

type TranslateResult struct {
	Text string
	OK   bool
}

type Translator interface {
	Translate(text, targetLang string) (string, error)
}
