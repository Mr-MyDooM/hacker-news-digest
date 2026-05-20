package hn

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mj/hacker-news-digest/config"
	"github.com/mj/hacker-news-digest/extractor"
)

type Parser struct {
	client *http.Client
}

func NewParser() *Parser {
	return &Parser{
		client: extractor.GetSafeClient(30 * time.Second),
	}
}

func (p *Parser) ParseNewsList() ([]*News, error) {
	resp, err := p.client.Get("https://news.ycombinator.com/")
	if err != nil {
		return nil, fmt.Errorf("fetch HN: %w", err)
	}
	defer resp.Body.Close()

	// Security: limit HN response to 1MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var newsList []*News
	doc.Find("tr.athing").Each(func(i int, sel *goquery.Selection) {
		news := &News{}

		titleSel := sel.Find("td.title .titleline > a").First()
		news.Title = strings.TrimSpace(titleSel.Text())
		news.URL, _ = titleSel.Attr("href")

		if news.Title == "" || news.URL == "" {
			return
		}

		rankSel := sel.Find("td.title span.rank")
		fmt.Sscanf(strings.TrimSpace(rankSel.Text()), "%d.", &news.Rank)

		subtext := sel.Next()
		scoreSel := subtext.Find("span.score")
		fmt.Sscanf(strings.TrimSpace(scoreSel.Text()), "%d points", &news.Score)

		authorSel := subtext.Find("a.hnuser")
		news.Author = strings.TrimSpace(authorSel.Text())

		ageSel := subtext.Find("span.age a")
		ageText := strings.TrimSpace(ageSel.Text())
		if ageText != "" {
			news.SubmitTime = parseHNAge(ageText)
		}

		comSel := subtext.Find("a").Last()
		comText := strings.TrimSpace(comSel.Text())
		if strings.Contains(comText, "comment") {
			fmt.Sscanf(comText, "%d", &news.CommentCount)
			href, _ := comSel.Attr("href")
			if href != "" {
				news.CommentURL = "https://news.ycombinator.com/" + href
			}
		}

		newsList = append(newsList, news)
	})

	return newsList, nil
}

func parseHNAge(text string) time.Time {
	now := time.Now().In(config.IST)
	parts := strings.Split(text, " ")
	if len(parts) < 2 {
		return now
	}

	var n int
	fmt.Sscanf(parts[0], "%d", &n)
	unit := parts[1]

	switch {
	case strings.HasPrefix(unit, "minute"):
		return now.Add(-time.Duration(n) * time.Minute)
	case strings.HasPrefix(unit, "hour"):
		return now.Add(-time.Duration(n) * time.Hour)
	case strings.HasPrefix(unit, "day"):
		return now.Add(-time.Duration(n) * 24 * time.Hour)
	case strings.HasPrefix(unit, "month"):
		return now.Add(-time.Duration(n) * 30 * 24 * time.Hour)
	case strings.HasPrefix(unit, "year"):
		return now.Add(-time.Duration(n) * 365 * 24 * time.Hour)
	}
	return now
}
