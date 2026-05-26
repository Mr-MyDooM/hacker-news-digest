package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/mj/hacker-news-digest/extractor"
)

const freeModelCacheTTL = 6 * time.Hour

type freeModelDiskCache struct {
	FetchedAt time.Time `json:"fetched_at"`
	Models    []string  `json:"models"`
}

func freeModelCachePath() string {
	dir := os.Getenv("OUTPUT_DIR")
	if dir == "" {
		dir = "output"
	}
	return dir + "/.openrouter_models_cache.json"
}

type openRouterModel struct {
	ID      string `json:"id"`
	Pricing struct {
		Prompt     string `json:"prompt"`
		Completion string `json:"completion"`
	} `json:"pricing"`
	ContextLength int `json:"context_length"`
}

type openRouterModelsResp struct {
	Data []openRouterModel `json:"data"`
}

// FetchFreeModels returns free model IDs from OpenRouter. Results are cached
// to disk for 6 hours — the binary exits after each 30-min run, so a file
// cache avoids redundant API calls across runs.
func FetchFreeModels(apiKey string) ([]string, error) {
	cachePath := freeModelCachePath()

	// Try reading valid disk cache first.
	if data, err := os.ReadFile(cachePath); err == nil {
		var cache freeModelDiskCache
		if json.Unmarshal(data, &cache) == nil && time.Since(cache.FetchedAt) < freeModelCacheTTL {
			log.Printf("OpenRouter: using disk-cached %d free models (age %s)", len(cache.Models), time.Since(cache.FetchedAt).Round(time.Minute))
			return cache.Models, nil
		}
	}

	ids, err := fetchFreeModelsFromAPI(apiKey)
	if err != nil {
		// Return stale cache rather than failing completely.
		if data, readErr := os.ReadFile(cachePath); readErr == nil {
			var cache freeModelDiskCache
			if json.Unmarshal(data, &cache) == nil && len(cache.Models) > 0 {
				log.Printf("OpenRouter: API error (%v), using stale cache (%d models)", err, len(cache.Models))
				return cache.Models, nil
			}
		}
		return nil, err
	}

	// Write fresh cache.
	if data, err := json.Marshal(freeModelDiskCache{FetchedAt: time.Now(), Models: ids}); err == nil {
		_ = os.WriteFile(cachePath, data, 0600)
	}

	return ids, nil
}

// fetchFreeModelsFromAPI discovers free models available on OpenRouter.
// Security: Uses extractor.GetSafeClient for SSRF protection and connection pooling.
func fetchFreeModelsFromAPI(apiKey string) ([]string, error) {
	client := extractor.GetSafeClient(15 * time.Second)
	req, err := http.NewRequest("GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch models: %w", err)
	}
	defer resp.Body.Close()

	// Security: limit response to 2MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var modelsResp openRouterModelsResp
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	var free []openRouterModel
	for _, m := range modelsResp.Data {
		if strings.HasSuffix(m.ID, ":free") &&
			m.Pricing.Prompt == "0" &&
			m.Pricing.Completion == "0" {
			free = append(free, m)
		}
	}

	// Prefer models with larger context windows (better for summarization).
	sort.Slice(free, func(i, j int) bool {
		return free[i].ContextLength > free[j].ContextLength
	})

	ids := make([]string, len(free))
	for i, m := range free {
		ids[i] = m.ID
	}
	log.Printf("OpenRouter: found %d free models", len(ids))
	return ids, nil
}
