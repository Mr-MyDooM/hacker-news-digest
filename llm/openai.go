package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mj/hacker-news-digest/db"
	"github.com/mj/hacker-news-digest/extractor"
)

type OpenAIClient struct {
	apiKey      string
	baseURL     string
	models      []string
	client      *http.Client
	rateLimiter *RateLimiter
}

// NewOpenAIClient creates a client for OpenAI or OpenRouter.
// Security: Uses extractor.GetSafeClient for SSRF protection and connection pooling.
func NewOpenAIClient(apiKey, baseURL string, models ...string) *OpenAIClient {
	return &OpenAIClient{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		models:  models,
		client:  extractor.GetSafeClient(60 * time.Second),
	}
}

func (c *OpenAIClient) SetRateLimit(rpm int) {
	c.rateLimiter = NewRateLimiter(rpm)
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model            string        `json:"model"`
	Messages         []chatMessage `json:"messages"`
	Temperature      float64       `json:"temperature"`
	FrequencyPenalty float64       `json:"frequency_penalty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *OpenAIClient) Summarize(content string) (string, db.Model, error) {
	messages := []chatMessage{
		{Role: "system", Content: summarizeSystemPrompt},
		{Role: "user", Content: truncateContent(content, 10000)},
	}

	result, err := c.callWithFallback(messages)
	if err != nil {
		return "", "", err
	}
	return cleanSummary(result), db.ModelOpenAI, nil
}

func (c *OpenAIClient) Translate(text, targetLang string) (string, error) {
	sysPrompt := fmt.Sprintf("You are a translator. Translate the following text to %s. Return only the translation, nothing else.", targetLang)
	messages := []chatMessage{
		{Role: "system", Content: sysPrompt},
		{Role: "user", Content: text},
	}
	return c.callWithFallback(messages)
}

func (c *OpenAIClient) callWithFallback(messages []chatMessage) (string, error) {
	var lastErr error
	for _, model := range c.models {
		result, err := c.call(chatRequest{
			Model:            model,
			Messages:         messages,
			Temperature:      0,
			FrequencyPenalty: 1,
		})
		if err == nil {
			return result, nil
		}
		lastErr = fmt.Errorf("model %s: %w", model, err)
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("no models configured")
}

// call performs the API request to OpenAI/OpenRouter.
// Security: Limits response size to prevent DoS via memory exhaustion.
func (c *OpenAIClient) call(req chatRequest) (string, error) {
	if c.rateLimiter != nil {
		c.rateLimiter.Wait()
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	// Security: limit response to 2MB
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var cr chatResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if cr.Error != nil {
		return "", fmt.Errorf("api error: %s", cr.Error.Message)
	}

	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return strings.TrimSpace(cr.Choices[0].Message.Content), nil
}

func truncateContent(content string, maxRunes int) string {
	runes := []rune(content)
	if len(runes) <= maxRunes {
		return content
	}
	return string(runes[:maxRunes]) + "\n\n...[truncated]"
}
