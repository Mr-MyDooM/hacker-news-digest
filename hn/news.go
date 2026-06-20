package hn

import (
	"encoding/json"
	"log"
	"strings"
	"unicode"

	"github.com/mj/hacker-news-digest/config"
	"github.com/mj/hacker-news-digest/db"
	"github.com/mj/hacker-news-digest/extractor"
	"github.com/mj/hacker-news-digest/llm"
)

var (
	cfg              *config.Config
	openAIClient     *llm.OpenAIClient
	geminiClient     *llm.GeminiClient
	openRouterClient *llm.OpenAIClient

	// Performance: pre-lowercased patterns to avoid redundant work in isBlockedContent.
	blockedPatterns = []string{
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
)

func init() {
	for i, p := range blockedPatterns {
		blockedPatterns[i] = strings.ToLower(p)
	}
}

func Init(c *config.Config) {
	cfg = c
	if c.OpenAIAPIKey != "" {
		openAIClient = llm.NewOpenAIClient(c.OpenAIAPIKey, c.OpenAIBase, c.OpenAIModel)
		openAIClient.SetRateLimit(c.LLMRate)
	}
	if c.GeminiAPIKey != "" && !c.DisableGemini {
		geminiClient = llm.NewGeminiClient(c.GeminiAPIKey, c.GeminiModel)
		geminiClient.SetRateLimit(c.LLMRate)
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
			openRouterClient.SetRateLimit(c.LLMRate)
		}
	}
}

func PrefetchSummaries(newsList []*News) {
	if len(newsList) == 0 {
		return
	}

	urls := make([]string, 0, len(newsList))
	for _, n := range newsList {
		if !n.IsHiringJob() && n.URL != "" {
			urls = append(urls, n.URL)
		}
	}

	summaries, err := db.GetSummaries(urls)
	if err != nil {
		log.Printf("Batch cache error: %v", err)
		return
	}

	for _, n := range newsList {
		if s, ok := summaries[n.URL]; ok {
			n.Cache = s
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

	// Optimization: Check cache FIRST to avoid expensive network I/O
	cached := n.Cache
	if cached == nil {
		var err error
		cached, err = db.GetSummary(n.URL)
		if err != nil {
			log.Printf("Cache error for %s: %v", n.URL, err)
		}
		n.Cache = cached
	}

	if cached != nil && cached.Summary != "" {
		// If cached summary is garbage (hallucinated, full of noise), force re-extraction
		if isGarbageSummary(cached.Summary) {
			log.Printf("Cache hit for %s but summary is garbage, re-extracting", n.URL)
		} else if cached.Model.IsFinal() || n.Score < cfg.OpenRouterScore {
			n.Summary = cached.Summary
			n.SummarizedBy = cached.Model
			// Restore cached image if we don't already have one
			if n.Image == nil && cached.ImageJSON.Valid && cached.ImageJSON.String != "" {
				var img extractor.WebImage
				if json.Unmarshal([]byte(cached.ImageJSON.String), &img) == nil && img.URL != "" {
					n.Image = &img
				}
			}
			log.Printf("Cache hit for %s (skipping fetch)", n.URL)
			return
		} else {
			log.Printf("Cache hit for %s, but model %s needs LLM upgrade", n.URL, cached.Model)
		}
	}

	result, err := extractor.Extract(n.URL, 65536)
	if err != nil {
		log.Printf("Failed to fetch %s: %v, using title as summary", n.URL, err)
		n.Summary = n.Title
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
		if jinaResult, jinaErr := extractor.ExtractViaJina(n.URL, 65536); jinaErr == nil && jinaResult.Content != "" {
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

	// Fetch image from candidates (meta image first, then body images)
	if cfg.ImageDir != "" && n.Image == nil {
		for _, imgURL := range result.Images {
			if imgURL == "" {
				continue
			}
			img, err := extractor.FetchImage(imgURL, n.URL, cfg.ImageDir)
			if err == nil {
				n.Image = img
				saveImageToCache(n)
				break
			}
			log.Printf("Failed to fetch image candidate %s for %s: %v", imgURL, n.URL, err)
		}
	}

	n.Summarize(n.Content)
}

// isBlockedContent detects login walls, JS-required pages, paywalls, WAF blocks, and error pages.
// Performance: Uses pre-lowercased blockedPatterns to avoid redundant allocations.
func isBlockedContent(content string) bool {
	lower := strings.ToLower(content)
	for _, p := range blockedPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// isGarbageSummary detects cached summaries that are LLM hallucinations or noise.
// These should force re-extraction rather than being treated as final.
func isGarbageSummary(summary string) bool {
	if len(summary) < 20 {
		return false
	}
	// Very long summaries (>500 chars) are suspicious — LLM summaries should be concise
	if len(summary) > 500 {
		return true
	}
	// Check for repeated words: "word" repeated 10+ times in a short span
	words := strings.Fields(summary)
	if len(words) > 0 {
		wordFreq := make(map[string]int, len(words))
		for _, w := range words {
			w = strings.Trim(w, ".,!?;:'\"()[]{}")
			if len(w) > 1 {
				wordFreq[w]++
			}
		}
		// If a single word appears >20% of all words, it's likely hallucinated repetition
		for _, count := range wordFreq {
			if len(words) > 20 && count > len(words)/5 {
				return true
			}
		}
	}
	// Low unique word ratio (repetitive garbage)
	unique := make(map[string]bool)
	for _, w := range words {
		unique[strings.ToLower(w)] = true
	}
	if len(words) > 30 && float64(len(unique))/float64(len(words)) < 0.5 {
		return true
	}
	// Low alphabetic ratio — too many symbols/noise characters
	alpha := 0
	total := 0
	for _, r := range summary {
		if r == ' ' || r == '\n' {
			continue
		}
		total++
		if unicode.IsLetter(r) {
			alpha++
		}
	}
	if total > 0 && float64(alpha)/float64(total) < 0.70 {
		return true
	}
	// Code-like artifacts in summary (never valid in a natural language summary)
	codePatterns := []string{"//", "{", "}", "=>", "->", "|", "!!", "??", ").", ");", "= "}
	matches := 0
	for _, p := range codePatterns {
		if strings.Contains(summary, p) {
			matches++
		}
	}
	if matches >= 3 {
		return true
	}
	return false
}

func (n *News) Summarize(content string) {
	// Short content doesn't need LLM summarization.
	if len([]rune(content)) <= cfg.SummarySize {
		n.Summary = content
		n.SummarizedBy = db.ModelPrefix
		db.PutSummary(&db.Summary{URL: n.URL, Summary: content, Model: db.ModelPrefix})
		return
	}

	var err error

	// n.Cache might have been populated in PullContent()
	cached := n.Cache
	if cached == nil {
		cached, err = db.GetSummary(n.URL)
		if err != nil {
			log.Printf("Cache error for %s: %v", n.URL, err)
		}
		n.Cache = cached
	}

	if cached != nil && cached.Summary != "" {
		// If cached model is non-final (e.g. prefix from a rate-limited run)
		// and score warrants LLM summarization, retry.
		if !cached.Model.IsFinal() && n.Score >= cfg.OpenRouterScore {
			log.Printf("Cache hit for %s, model %s (non-final, will retry LLM)", n.URL, cached.Model)
		} else {
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
	}

	var summary string
	var model db.Model

	switch {
	case n.Score >= cfg.OpenRouterScore && openRouterClient != nil:
		summary, model, err = openRouterClient.Summarize(content)
		if err == nil {
			break
		}
		log.Printf("OpenRouter failed for %s: %v", n.URL, err)
		fallthrough

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

	default:
		summary = prefixSummary(content, cfg.SummarySize)
		model = db.ModelPrefix
	}

	// Safety net: if LLM summary is still garbage (happens for stubborn pages),
	// fall back to prefix to avoid caching trash.
	if model != db.ModelPrefix && isGarbageSummary(summary) {
		log.Printf("LLM summary for %s is garbage, falling back to prefix", n.URL)
		summary = prefixSummary(content, cfg.SummarySize)
		model = db.ModelPrefix
	}

	n.Summary = summary
	n.SummarizedBy = model

	if summary != "" {
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
