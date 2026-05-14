package hn

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const algoliaURL = "https://hn.algolia.com/api/v1/search_by_date"

type algoliaResponse struct {
	Hits    []algoliaHit `json:"hits"`
	NbPages int          `json:"nbPages"`
}

type algoliaHit struct {
	ObjectID    string `json:"objectID"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Points      int    `json:"points"`
	Author      string `json:"author"`
	CreatedAt   string `json:"created_at"`
	NumComments int    `json:"num_comments"`
	AskHN       bool   `json:"ask_hn"`
}

func GetDailyNews(updatableDays int) (map[string][]*News, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	seen := make(map[string]bool)
	byDate := make(map[string][]*News)
	threshold := time.Now().Add(-time.Duration(updatableDays)*24*time.Hour).Unix()

	for page := 0; page < 50; page++ {
		// Performance: Use server-side filtering and limit pages to reduce API overhead.
		u := fmt.Sprintf("%s?tags=front_page&hitsPerPage=200&page=%d&numericFilters=created_at_i%s%d", algoliaURL, page, url.QueryEscape(">"), threshold)
		log.Printf("Fetching Algolia page %d: %s", page, u)
		resp, err := client.Get(u)
		if err != nil {
			return nil, fmt.Errorf("algolia page %d: %w", page, err)
		}
		// Security: limit Algolia API response to 2MB
		body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("algolia page %d read: %w", page, err)
		}

		var ar algoliaResponse
		if err := json.Unmarshal(body, &ar); err != nil {
			return nil, fmt.Errorf("parse algolia: %w", err)
		}
		if len(ar.Hits) == 0 {
			break
		}

		for _, hit := range ar.Hits {
			if seen[hit.ObjectID] {
				continue
			}
			seen[hit.ObjectID] = true

			createdAt, _ := time.Parse(time.RFC3339, hit.CreatedAt)
			if time.Since(createdAt) > time.Duration(updatableDays)*24*time.Hour {
				// Optimization: Algolia search_by_date is strictly chronological.
				// If we hit an old story, all subsequent stories and pages are even older.
				return byDate, nil
			}

			newsURL := hit.URL
			if newsURL == "" {
				if hit.AskHN {
					newsURL = fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID)
				} else {
					continue
				}
			}

			story := &News{
				Title:        hit.Title,
				URL:          newsURL,
				Score:        hit.Points,
				Author:       hit.Author,
				SubmitTime:   createdAt,
				CommentCount: hit.NumComments,
				CommentURL:   fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID),
			}

			dateKey := createdAt.Format("2006-01-02")
			byDate[dateKey] = append(byDate[dateKey], story)
		}
	}

	return byDate, nil
}
