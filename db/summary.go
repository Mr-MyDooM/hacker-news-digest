package db

import (
	"database/sql"
	"fmt"
	"strings"
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

func GetSummaries(urls []string) (map[string]*Summary, error) {
	if cfg.DisableSummaryCache || len(urls) == 0 {
		return make(map[string]*Summary), nil
	}

	summaries := make(map[string]*Summary)

	// SQLite has a limit on parameters (usually 999), so we chunk the requests.
	const chunkSize = 900
	for i := 0; i < len(urls); i += chunkSize {
		end := i + chunkSize
		if end > len(urls) {
			end = len(urls)
		}
		chunk := urls[i:end]

		placeholders := make([]string, len(chunk))
		args := make([]interface{}, len(chunk))
		for j, u := range chunk {
			placeholders[j] = "?"
			args[j] = u
		}

		query := fmt.Sprintf(`SELECT url, summary, model, birth, access, favicon, image_name, image_json
			FROM summary WHERE url IN (%s)`, strings.Join(placeholders, ","))

		err := func() error {
			rows, err := DB.Query(query, args...)
			if err != nil {
				return fmt.Errorf("get summaries: %w", err)
			}
			defer rows.Close()

			for rows.Next() {
				s := &Summary{}
				var birth, access string
				err := rows.Scan(&s.URL, &s.Summary, &s.Model, &birth, &access, &s.Favicon, &s.ImageName, &s.ImageJSON)
				if err != nil {
					return fmt.Errorf("scan summary: %w", err)
				}
				s.Birth, _ = time.Parse("2006-01-02 15:04:05", birth)
				s.Access, _ = time.Parse("2006-01-02 15:04:05", access)
				summaries[s.URL] = s
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	return summaries, nil
}

func FilterURLs(urls []string) (map[string]bool, error) {
	found := make(map[string]bool)
	if len(urls) == 0 {
		return found, nil
	}

	// SQLite has a limit on parameters (usually 999), so we chunk the requests.
	const chunkSize = 900
	for i := 0; i < len(urls); i += chunkSize {
		end := i + chunkSize
		if end > len(urls) {
			end = len(urls)
		}
		chunk := urls[i:end]

		placeholders := make([]string, len(chunk))
		args := make([]interface{}, len(chunk))
		for j, u := range chunk {
			placeholders[j] = "?"
			args[j] = u
		}

		query := fmt.Sprintf(`SELECT url FROM summary WHERE url IN (%s)`, strings.Join(placeholders, ","))

		err := func() error {
			rows, err := DB.Query(query, args...)
			if err != nil {
				return fmt.Errorf("filter urls: %w", err)
			}
			defer rows.Close()

			for rows.Next() {
				var u string
				if err := rows.Scan(&u); err == nil {
					found[u] = true
				}
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	return found, nil
}
