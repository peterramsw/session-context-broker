// Package tokens provides token counting via Anthropic API and heuristic estimation.
package tokens

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	countTokensURL = "https://api.anthropic.com/v1/messages/count_tokens"
	anthropicModel = "claude-sonnet-4-6"
	apiVersion     = "2023-06-01"
	apiTimeout     = 30 * time.Second
)

// CJK Unicode ranges for token estimation.
const (
	cjkUnifiedStart    = 0x4E00
	cjkUnifiedEnd      = 0x9FFF
	cjkExtAStart       = 0x3400
	cjkExtAEnd         = 0x4DBF
	cjkTokenMultiplier = 1.5
	asciiTokenRatio    = 0.25
)

// EstimateTokens provides a rough token count using character-class heuristics.
// CJK characters are weighted at ~1.5 tokens each; other characters at ~0.25.
// This deliberately undercounts single-character inputs due to int truncation,
// which is acceptable for aggregated context-budget estimates.
func EstimateTokens(text string) int {
	cjkCount := 0
	otherCount := 0
	for _, ch := range text {
		if (ch >= cjkUnifiedStart && ch <= cjkUnifiedEnd) ||
			(ch >= cjkExtAStart && ch <= cjkExtAEnd) {
			cjkCount++
		} else {
			otherCount++
		}
	}
	return int(float64(cjkCount)*cjkTokenMultiplier + float64(otherCount)*asciiTokenRatio)
}

// CountTokensAPI calls the Anthropic count_tokens endpoint.
// Returns (tokenCount, nil) on success, or (0, error) on failure.
// Requires ANTHROPIC_API_KEY environment variable.
func CountTokensAPI(text string) (int, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return 0, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}
	return countTokens(text, apiKey, countTokensURL, &http.Client{Timeout: apiTimeout})
}

func countTokens(text string, apiKey string, endpoint string, client *http.Client) (int, error) {
	payload := map[string]interface{}{
		"model": anthropicModel,
		"messages": []map[string]interface{}{
			{"role": "user", "content": text},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", apiVersion)
	req.Header.Set("content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		InputTokens int `json:"input_tokens"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("parse response: %w", err)
	}
	return result.InputTokens, nil
}
