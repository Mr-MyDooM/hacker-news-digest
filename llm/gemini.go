package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/mj/hacker-news-digest/db"
)

const geminiModelCacheTTL = 6 * time.Hour

type geminiModelDiskCache struct {
	FetchedAt time.Time `json:"fetched_at"`
	Models    []string  `json:"models"`
}

func geminiModelCachePath() string {
	dir := os.Getenv("OUTPUT_DIR")
	if dir == "" {
		dir = "output"
	}
	return dir + "/.gemini_models_cache.json"
}

type geminiModelInfo struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Methods     []string `json:"supportedGenerationMethods"`
}

type geminiModelsResp struct {
	Models []geminiModelInfo `json:"models"`
}

// FetchGeminiModels returns available Gemini model IDs that support generateContent.
// Results are cached to disk for 6 hours.
func FetchGeminiModels(apiKey string) ([]string, error) {
	cachePath := geminiModelCachePath()

	if data, err := os.ReadFile(cachePath); err == nil {
		var cache geminiModelDiskCache
		if json.Unmarshal(data, &cache) == nil && time.Since(cache.FetchedAt) < geminiModelCacheTTL {
			log.Printf("Gemini: using disk-cached %d models (age %s)", len(cache.Models), time.Since(cache.FetchedAt).Round(time.Minute))
			return cache.Models, nil
		}
	}

	ids, err := fetchGeminiModelsFromAPI(apiKey)
	if err != nil {
		if data, readErr := os.ReadFile(cachePath); readErr == nil {
			var cache geminiModelDiskCache
			if json.Unmarshal(data, &cache) == nil && len(cache.Models) > 0 {
				log.Printf("Gemini: API error (%v), using stale cache (%d models)", err, len(cache.Models))
				return cache.Models, nil
			}
		}
		return nil, err
	}

	if data, err := json.Marshal(geminiModelDiskCache{FetchedAt: time.Now(), Models: ids}); err == nil {
		_ = os.WriteFile(cachePath, data, 0600)
	}

	return ids, nil
}

func fetchGeminiModelsFromAPI(apiKey string) ([]string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	u := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s", apiKey)
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("fetch models: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var modelsResp geminiModelsResp
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Filter for models that support generateContent and prefer flash models.
	var flash, other []string
	for _, m := range modelsResp.Models {
		hasGenerate := false
		for _, method := range m.Methods {
			if method == "generateContent" {
				hasGenerate = true
				break
			}
		}
		if !hasGenerate {
			continue
		}
		name := strings.TrimPrefix(m.Name, "models/")
		if strings.Contains(name, "flash") {
			flash = append(flash, name)
		} else {
			other = append(other, name)
		}
	}

	// Flash models first (free-tier friendly), then others.
	sort.Sort(sort.Reverse(sort.StringSlice(flash)))
	sort.Sort(sort.Reverse(sort.StringSlice(other)))

	ids := append(flash, other...)
	log.Printf("Gemini: found %d models (%d flash, %d other)", len(ids), len(flash), len(other))
	return ids, nil
}

type GeminiClient struct {
	apiKey      string
	models      []string
	client      *http.Client
	rateLimiter *RateLimiter
}

func NewGeminiClient(apiKey, model string) *GeminiClient {
	models := []string{model}
	// Try to discover available models. Fall back to hardcoded list on error.
	discovered, err := FetchGeminiModels(apiKey)
	if err != nil {
		log.Printf("Gemini model discovery failed: %v", err)
		// Hardcoded fallback if API call fails.
		discovered = []string{
			"gemini-2.5-flash",
			"gemini-2.0-flash",
			"gemini-1.5-flash",
		}
	}
	// Put the configured model first, then discovered models.
	seen := map[string]bool{model: true}
	models = []string{model}
	for _, m := range discovered {
		if !seen[m] {
			models = append(models, m)
			seen[m] = true
		}
	}

	return &GeminiClient{
		apiKey: apiKey,
		models: models,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *GeminiClient) SetRateLimit(rpm int) {
	c.rateLimiter = NewRateLimiter(rpm)
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents         []geminiContent `json:"contents"`
	SystemInstruction *geminiContent  `json:"system_instruction,omitempty"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const summarizeSystemPrompt = "Summarize the input in 2 concise English sentences. Do not exceed 200 characters. Do not start with 'This article', 'The article', 'This post', or 'The post'. Write in plain text, no Markdown. Respond with ONLY the two sentences. Do not include any reasoning, chain-of-thought, draft attempts, character counts, or constraint checks."

func (c *GeminiClient) Summarize(content string) (string, db.Model, error) {
	req := geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: summarizeSystemPrompt}},
		},
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: truncateContent(content, 15000)}}},
		},
	}

	result, err := c.call(req)
	if err != nil {
		return "", "", err
	}

	return cleanSummary(result), db.ModelGemini, nil
}

// cleanSummary strips markdown, model preambles, and chain-of-thought reasoning.
func cleanSummary(s string) string {
	s = strings.ReplaceAll(s, "**", "")

	// Strip chain-of-thought: if the response contains draft/attempt markers,
	// extract only the final 1-2 sentence summary from the end.
	if hasReasoningMarkers(s) {
		s = extractFinalSummary(s)
	}

	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)
	for _, p := range []string{"summary:", "here is a summary:", "here's a summary:"} {
		if strings.HasPrefix(lower, p) {
			s = s[len(p):]
			lower = lower[len(p):]
		}
	}
	if idx := strings.Index(s, "</think>"); idx >= 0 {
		s = s[idx+len("</think>"):]
	}
	s = strings.TrimLeftFunc(s, func(r rune) bool {
		return r != ' ' && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9')
	})
	return strings.TrimSpace(s)
}

// hasReasoningMarkers checks for Gemini chain-of-thought patterns.
func hasReasoningMarkers(s string) bool {
	markers := []string{
		"*Draft", "*Attempt",
		"Character count", "Sentence count",
		"Start check", "Formatting check",
		"Constraint", "forbidden start",
	}
	for _, m := range markers {
		if strings.Contains(s, m) {
			return true
		}
	}
	return false
}

// extractFinalSummary finds the last short paragraph that looks like a summary
// by stripping bullet-point reasoning and taking the final text.
func extractFinalSummary(s string) string {
	// Try final-text markers first
	finalMarkers := []string{
		"Final text:", "Final string:", "Final choice:",
		"Let's go with:", "Final Polish:", "Final draft:",
	}
	lastIdx := -1
	for _, m := range finalMarkers {
		if idx := strings.LastIndex(s, m); idx > lastIdx {
			lastIdx = idx + len(m)
		}
	}
	if lastIdx > 0 {
		s = s[lastIdx:]
	}

	// Remove remaining reasoning lines (bullet points, checks, meta-commentary)
	lines := strings.Split(s, "\n")
	var clean []string
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "*") || strings.HasPrefix(t, "-") || strings.HasPrefix(t, "•") {
			continue
		}
		skipPrefixes := []string{"wait,", "let's", "character", "sentence", "start", "formatting",
			"plain text", "markdown", "concise", "constraint", "draft", "attempt"}
		skip := false
		lt := strings.ToLower(t)
		for _, sp := range skipPrefixes {
			if strings.HasPrefix(lt, sp) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		clean = append(clean, t)
	}

	if len(clean) > 0 {
		return strings.Join(clean, " ")
	}
	return strings.TrimSpace(s)
}

func (c *GeminiClient) call(req geminiRequest) (string, error) {
	if c.rateLimiter != nil {
		c.rateLimiter.Wait()
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	var lastErr error
	for _, model := range c.models {
		u := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, c.apiKey)

		httpReq, err := http.NewRequest("POST", u, bytes.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := c.client.Do(httpReq)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)

		var gr geminiResponse
		if err := json.Unmarshal(respBody, &gr); err != nil {
			lastErr = err
			continue
		}

		if gr.Error != nil {
			lastErr = fmt.Errorf("gemini error: %s", gr.Error.Message)
			continue
		}

		if len(gr.Candidates) > 0 && len(gr.Candidates[0].Content.Parts) > 0 {
			return strings.TrimSpace(gr.Candidates[0].Content.Parts[0].Text), nil
		}
	}

	return "", fmt.Errorf("all gemini models failed: %w", lastErr)
}
