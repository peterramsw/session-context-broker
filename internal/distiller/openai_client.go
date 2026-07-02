// Package distiller calls an optional local OpenAI-compatible endpoint to turn
// filtered transcripts into structured handoff artifacts.
package distiller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/config"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Client struct {
	BaseURL     string
	APIKey      string
	Model       string
	Timeout     time.Duration
	HTTP        *http.Client
	UserAgent   string
	Temperature float64
	TopP        *float64
	TopK        int
}

func NewClient(cfg config.LocalLLMConfig) Client {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return Client{
		BaseURL:     strings.TrimRight(cfg.BaseURL, "/"),
		APIKey:      cfg.APIKey,
		Model:       cfg.Model,
		Timeout:     timeout,
		HTTP:        &http.Client{Timeout: timeout},
		Temperature: temperatureOrDefault(cfg.Temperature),
		TopP:        cfg.TopP,
		TopK:        cfg.TopK,
	}
}

func (c Client) Chat(ctx context.Context, messages []Message, maxOutputTokens int) (string, error) {
	if c.BaseURL == "" {
		return "", fmt.Errorf("local_llm.base_url is required")
	}
	if c.Model == "" {
		return "", fmt.Errorf("local_llm.model is required")
	}
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: c.Timeout}
	}
	if maxOutputTokens <= 0 {
		maxOutputTokens = 4096
	}
	payload := map[string]any{
		"model":       c.Model,
		"messages":    messages,
		"temperature": c.Temperature,
		"max_tokens":  maxOutputTokens,
		"stream":      false,
	}
	if c.TopP != nil {
		payload["top_p"] = *c.TopP
	}
	if c.TopK > 0 {
		payload["top_k"] = c.TopK
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal local LLM request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create local LLM request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	if c.UserAgent != "" {
		req.Header.Set("user-agent", c.UserAgent)
	}
	if c.APIKey != "" {
		req.Header.Set("authorization", "Bearer "+c.APIKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("local LLM request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read local LLM response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("local LLM returned status %d: %s", resp.StatusCode, string(respBody))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("parse local LLM response: %w", err)
	}
	if len(parsed.Choices) == 0 || parsed.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("local LLM response had no message content")
	}
	return parsed.Choices[0].Message.Content, nil
}

func temperatureOrDefault(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
