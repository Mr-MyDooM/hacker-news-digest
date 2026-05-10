package hn

import (
	"time"

	"github.com/mj/hacker-news-digest/db"
	"github.com/mj/hacker-news-digest/extractor"
)

type News struct {
	Rank         int
	Title        string
	URL          string
	Score        int
	Author       string
	SubmitTime   time.Time
	CommentCount int
	CommentURL   string
	Content      string
	Summary      string
	SummarizedBy db.Model
	Image        *extractor.WebImage
	Favicon      string
	Cache        *db.Summary
}

func (n *News) Slug() string {
	slug := make([]byte, 0, len(n.Title))
	for _, c := range n.Title {
		switch {
		case c >= 'a' && c <= 'z' || c >= '0' && c <= '9':
			slug = append(slug, byte(c))
		case c >= 'A' && c <= 'Z':
			slug = append(slug, byte(c+32))
		case c == ' ' || c == '-':
			slug = append(slug, '-')
		}
	}
	if len(slug) > 80 {
		slug = slug[:80]
	}
	slug = trimHyphen(slug)
	return string(slug)
}

func trimHyphen(b []byte) []byte {
	for len(b) > 0 && b[0] == '-' {
		b = b[1:]
	}
	for len(b) > 0 && b[len(b)-1] == '-' {
		b = b[:len(b)-1]
	}
	return b
}

func (n *News) GetScore() int {
	return n.Score
}

func (n *News) IsHiringJob() bool {
	return n.Title == "Ask HN: Who is hiring?" ||
		n.Title == "Ask HN: Who wants to be hired?" ||
		n.Title == "Ask HN: Freelancer? Seeking freelancer?"
}
