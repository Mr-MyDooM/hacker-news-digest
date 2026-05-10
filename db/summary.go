package db

import (
	"database/sql"
	"fmt"
	"time"
)

func GetSummary(url string) (*Summary, error) {
	if cfg.DisableSummaryCache {
		return &Summary{URL: url}, nil
	}
	s := &Summary{}
	var birth, access string
	err := DB.QueryRow(`SELECT url, summary, model, birth, access, favicon, image_name, image_json
		FROM summary WHERE url = ?`, url).Scan(
		&s.URL, &s.Summary, &s.Model, &birth, &access, &s.Favicon, &s.ImageName, &s.ImageJSON)
	if err == sql.ErrNoRows {
		return &Summary{URL: url}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get summary: %w", err)
	}
	s.Birth, _ = time.Parse("2006-01-02 15:04:05", birth)
	s.Access, _ = time.Parse("2006-01-02 15:04:05", access)
	return s, nil
}

func PutSummary(s *Summary) error {
	_, err := DB.Exec(`INSERT INTO summary (url, summary, model, birth, access, favicon, image_name, image_json)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?, ?)
		ON CONFLICT(url) DO UPDATE SET
			summary=excluded.summary,
			model=excluded.model,
			access=CURRENT_TIMESTAMP,
			favicon=COALESCE(excluded.favicon, summary.favicon),
			image_name=COALESCE(excluded.image_name, summary.image_name),
			image_json=COALESCE(excluded.image_json, summary.image_json)`,
		s.URL, s.Summary, s.Model, s.Favicon, s.ImageName, s.ImageJSON)
	if err != nil {
		return fmt.Errorf("put summary: %w", err)
	}
	return nil
}

func FilterURLs(urls []string) (map[string]bool, error) {
	found := make(map[string]bool)
	for _, u := range urls {
		var count int
		err := DB.QueryRow(`SELECT COUNT(*) FROM summary WHERE url = ?`, u).Scan(&count)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			found[u] = true
		}
	}
	return found, nil
}
