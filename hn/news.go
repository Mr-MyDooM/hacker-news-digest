package hn

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/mj/hacker-news-digest/config"
	"github.com/mj/hacker-news-digest/db"
	"github.com/mj/hacker-news-digest/extractor"
	"github.com/mj/hacker-news-digest/llm"
)

var (
	cfg             *config.Config
	openAIClient    *llm.OpenAIClient
	geminiClient    *llm.GeminiClient
	openRouterClient *llm.OpenAIClient
)

func Init(c *config.Config) {
	cfg = c
	if c.OpenAIAPIKey != "" {
		openAIClient = llm.NewOpenAIClient(c.OpenAIAPIKey, c.OpenAIBase, c.OpenAIModel)
	}
	if c.GeminiAPIKey != "" && !c.DisableGemini {
		geminiClient = llm.NewGeminiClient(c.GeminiAPIKey, c.GeminiModel)
	}
	if c.OpenRouterAPIKey != "" && !c.DisableOpenRouter {
		models := c.OpenRouterModels
		if len(models) == 0 {
			// Dynamically discover free models from OpenRouter API.
			discovered, err := llm.FetchFreeModels(c.OpenRouterAPIKey)
			if err != nil {
				log.Printf("OpenRouter model discovery failed: %v", err)
			} else {
				models = discovered
			}
		}
		if len(models) > 0 {
			openRouterClient = llm.NewOpenAIClient(c.OpenRouterAPIKey, "https://openrouter.ai/api/v1", models...)
			openRouterClient.SetRateLimit(2 * time.Second)
		}
	}
}

func (n *News) PullContent() {
	if n.IsHiringJob() {
		n.Content = n.Title
		n.Summary = "Hiring thread"
		n.SummarizedBy = db.ModelPrefix
		return
	}

	result, err := extractor.Extract(n.URL, cfg.SummarySize*3)
	if err != nil {
		log.Printf("Failed to fetch %s: %v", n.URL, err)
		n.Summary = ""
		n.SummarizedBy = db.ModelPrefix
		return
	}

	n.Content = result.Content
	if n.Content == "" {
		n.Content = result.Description
	}
	if n.Content == "" {
		n.Content = result.Title
	}

	if isBlockedContent(n.Content) {
		log.Printf("Blocked content for %s, retrying via Jina", n.URL)
		if jinaResult, jinaErr := extractor.ExtractViaJina(n.URL, cfg.SummarySize*3); jinaErr == nil && jinaResult.Content != "" {
			n.Content = jinaResult.Content
			log.Printf("Jina succeeded for %s", n.URL)
		} else {
			if jinaErr != nil {
				log.Printf("Jina also failed for %s: %v", n.URL, jinaErr)
			}
			n.Summary = n.Title
			n.SummarizedBy = db.ModelPrefix
			return
		}
	}

	// Fetch og:image when available
	if result.Image != "" && cfg.ImageDir != "" && n.Image == nil {
		img, err := extractor.FetchImage(result.Image, n.URL, cfg.ImageDir)
		if err == nil {
			n.Image = img
			saveImageToCache(n)
		}
	}

	n.Summarize(n.Content)
}

// isBlockedContent detects login walls, JS-required pages, paywalls, WAF blocks, and error pages
func isBlockedContent(content string) bool {
	patterns := []string{
		"Something went wrong",
		"privacy related extensions",
		"Please disable them and try again",
		"Sign in to continue",
		"Log in to Twitter",
		"Subscribe to continue reading",
		"This content is for subscribers",
		"Access denied",
		"Please enable JavaScript",
		"Please enable JS",
		"JavaScript is required",
		"requires JavaScript to",
		"enable javascript",
		"disable any ad blocker",
		// YouTube footer fingerprint — page loaded but JS content missing
		"AboutPressCopyrightContact usCreators",
		// WAF / CDN block pages
		"forbidden",
		"you don't have permission",
		"access to this page is forbidden",
		"i challenge thee",
		"attention required",
		"checking your browser",
		"just a moment",
		"ddos protection",
	}
	lower := strings.ToLower(content)
	for _, p := range patterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

func (n *News) Summarize(content string) {
	// Short content doesn't need LLM summarization.
	if len([]rune(content)) <= cfg.SummarySize {
		n.Summary = content
		n.SummarizedBy = db.ModelPrefix
		return
	}

	cached, err := db.GetSummary(n.URL)
	if err != nil {
		log.Printf("Cache error for %s: %v", n.URL, err)
	}
	n.Cache = cached

	if cached != nil && cached.Summary != "" {
		n.Summary = cached.Summary
		n.SummarizedBy = cached.Model
		// Restore cached image if we don't already have one
		if n.Image == nil && cached.ImageJSON.Valid && cached.ImageJSON.String != "" {
			var img extractor.WebImage
			if json.Unmarshal([]byte(cached.ImageJSON.String), &img) == nil && img.URL != "" {
				n.Image = &img
			}
		}
		// Persist image into cache if we just fetched one and cache didn't have it
		if n.Image != nil && (!cached.ImageJSON.Valid || cached.ImageJSON.String == "") {
			if data, err := json.Marshal(n.Image); err == nil {
				cached.ImageJSON.String = string(data)
				cached.ImageJSON.Valid = true
				cached.ImageName.String = imageFilename(n.Image.URL)
				cached.ImageName.Valid = true
				if err := db.PutSummary(cached); err != nil {
					log.Printf("Failed to update image cache for %s: %v", n.URL, err)
				}
			}
		}
		log.Printf("Cache hit for %s, model %s", n.URL, cached.Model)
		return
	}

	var summary string
	var model db.Model

	switch {
	case n.Score >= cfg.OpenAIScore && openAIClient != nil:
		summary, model, err = openAIClient.Summarize(content)
		if err == nil {
			break
		}
		log.Printf("OpenAI failed for %s: %v", n.URL, err)
		fallthrough

	case n.Score >= cfg.GeminiScore && geminiClient != nil:
		summary, model, err = geminiClient.Summarize(content)
		if err == nil {
			break
		}
		log.Printf("Gemini failed for %s: %v", n.URL, err)
		fallthrough

	case n.Score >= cfg.LocalLLMScore && openRouterClient != nil:
		summary, model, err = openRouterClient.Summarize(content)
		if err == nil {
			break
		}
		log.Printf("OpenRouter failed for %s: %v", n.URL, err)
		fallthrough

	default:
		summary = prefixSummary(content, cfg.SummarySize)
		model = db.ModelPrefix
	}

	n.Summary = summary
	n.SummarizedBy = model

	if model != db.ModelPrefix && summary != "" {
		entry := &db.Summary{
			URL:     n.URL,
			Summary: summary,
			Model:   model,
		}
		if n.Image != nil {
			if data, err := json.Marshal(n.Image); err == nil {
				entry.ImageJSON.String = string(data)
				entry.ImageJSON.Valid = true
				entry.ImageName.String = imageFilename(n.Image.URL)
				entry.ImageName.Valid = true
			}
		}
		if err := db.PutSummary(entry); err != nil {
			log.Printf("Failed to cache summary for %s: %v", n.URL, err)
		}
	}
}

func imageFilename(servedURL string) string {
	parts := strings.Split(servedURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func saveImageToCache(n *News) {
	entry := &db.Summary{URL: n.URL}
	if n.Image != nil {
		if data, err := json.Marshal(n.Image); err == nil {
			entry.ImageJSON.String = string(data)
			entry.ImageJSON.Valid = true
			entry.ImageName.String = imageFilename(n.Image.URL)
			entry.ImageName.Valid = true
		}
	}
	if err := db.PutSummary(entry); err != nil {
		log.Printf("Failed to cache image for %s: %v", n.URL, err)
	}
}

func prefixSummary(content string, maxLen int) string {
	runes := []rune(strings.TrimSpace(content))
	if len(runes) <= maxLen {
		return string(runes)
	}
	return string(runes[:maxLen]) + " ..."
}

func (n *News) TranslateSummary() {
	// Chinese translation via OpenAI
	if cfg.DisableTranslation || openAIClient == nil {
		return
	}
	if db.TranslationExists(n.Summary, "zh") {
		return
	}
	trans, err := openAIClient.Translate(n.Summary, "Chinese")
	if err != nil {
		log.Printf("Translation failed for %s: %v", n.URL, err)
		return
	}
	db.AddTranslation(n.Summary, trans, "zh")
}
