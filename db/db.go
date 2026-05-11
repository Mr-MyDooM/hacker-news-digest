package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"

	"github.com/mj/hacker-news-digest/config"
)

var cfg *config.Config

func Init(c *config.Config) error {
	cfg = c
	var err error
	DB, err = sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	DB.SetMaxOpenConns(1)
	DB.SetMaxIdleConns(1)

	if err = createTables(); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}
	return nil
}

func createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS summary (
			url TEXT PRIMARY KEY,
			summary TEXT NOT NULL DEFAULT '',
			model TEXT NOT NULL DEFAULT 'Full',
			birth TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			access TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			favicon TEXT,
			image_name TEXT,
			image_json TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_summary_image_name ON summary(image_name)`,
		`CREATE TABLE IF NOT EXISTS translation (
			source TEXT PRIMARY KEY,
			target TEXT NOT NULL DEFAULT '',
			language TEXT NOT NULL DEFAULT '',
			access TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

type Model string

const (
	ModelPrefix      Model = "Prefix"
	ModelFull        Model = "Full"
	ModelEmbed       Model = "Embed"
	ModelTransformer Model = "GoogleT5"
	ModelLLaMA       Model = "Llama"
	ModelStep        Model = "Step"
	ModelGemma       Model = "Gemma"
	ModelQwen        Model = "Qwen"
	ModelOpenAI      Model = "OpenAI"
	ModelGemini      Model = "Gemini"
	ModelOpenRouter  Model = "OpenRouter"
)

func (m Model) IsPrefix() bool { return m == ModelPrefix }

func (m Model) CanTruncate() bool {
	return m != ModelOpenAI && m != ModelEmbed
}

func (m Model) IsFinal() bool {
	switch m {
	case ModelEmbed, ModelOpenAI, ModelGemini, ModelOpenRouter,
		ModelGemma, ModelLLaMA, ModelStep, ModelQwen:
		return true
	}
	return false
}

func (m Model) NeedEscape() bool {
	return false
}

type Summary struct {
	URL       string
	Summary   string
	Model     Model
	Birth     time.Time
	Access    time.Time
	Favicon   sql.NullString
	ImageName sql.NullString
	ImageJSON sql.NullString
}

func ExpireSummaries() (int, error) {
	cutoff := time.Now().Add(-time.Duration(cfg.SummaryTTL) * time.Second)
	result, err := DB.Exec(`DELETE FROM summary WHERE access < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("expire summaries: %w", err)
	}
	n, _ := result.RowsAffected()
	log.Printf("evicted %d summary items", n)

	contentCutoff := time.Now().Add(-24 * time.Hour)
	result2, err := DB.Exec(
		`DELETE FROM summary WHERE access < ? AND model IN (?, ?, ?)`,
		contentCutoff, ModelPrefix, ModelFull, ModelEmbed)
	if err != nil {
		return 0, fmt.Errorf("expire content: %w", err)
	}
	n2, _ := result2.RowsAffected()
	log.Printf("evicted %d full content items", n2)

	return int(n + n2), nil
}

var DB *sql.DB
