package db

import (
	"database/sql"
	"fmt"
	"time"
)

type Translation struct {
	Source   string
	Target   string
	Language string
	Access   time.Time
}

func GetTranslation(text, lang string) (*Translation, error) {
	t := &Translation{}
	var access string
	err := DB.QueryRow(`SELECT source, target, language, access FROM translation WHERE source = ? AND language = ?`,
		text, lang).Scan(&t.Source, &t.Target, &t.Language, &access)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get translation: %w", err)
	}
	t.Access, _ = time.Parse("2006-01-02 15:04:05", access)
	return t, nil
}

func TranslationExists(text, lang string) bool {
	var count int
	DB.QueryRow(`SELECT COUNT(*) FROM translation WHERE source = ? AND language = ?`, text, lang).Scan(&count)
	return count > 0
}

func AddTranslation(source, target, lang string) error {
	_, err := DB.Exec(`INSERT INTO translation (source, target, language, access)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(source, language) DO UPDATE SET
			target=excluded.target,
			access=CURRENT_TIMESTAMP`, source, target, lang)
	if err != nil {
		return fmt.Errorf("add translation: %w", err)
	}
	return nil
}

func ExpireTranslations() {
	DB.Exec(`DELETE FROM translation WHERE access < datetime('now', '-30 days')`)
}
