package db

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

func ExpireImages() error {
	entries, err := os.ReadDir(cfg.ImageDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read image dir: %w", err)
	}

	rand.Shuffle(len(entries), func(i, j int) {
		entries[i], entries[j] = entries[j], entries[i]
	})

	maxCheck := 1000
	if len(entries) < maxCheck {
		maxCheck = len(entries)
	}

	removed := 0
	for _, e := range entries[:maxCheck] {
		if e.IsDir() {
			continue
		}
		var count int
		err := DB.QueryRow(`SELECT COUNT(*) FROM summary WHERE image_name = ?`, e.Name()).Scan(&count)
		if err != nil {
			continue
		}
		if count == 0 {
			os.Remove(filepath.Join(cfg.ImageDir, e.Name()))
			removed++
		}
	}
	log.Printf("removed %d/%d feature images", removed, maxCheck)
	return nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
