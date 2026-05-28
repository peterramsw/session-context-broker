package tokens

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int
	}{
		{
			// 11 ASCII chars, ~0.25 tokens each -> heuristic approximation: 2
			name: "pure ASCII",
			text: "hello world",
			want: 2,
		},
		{
			name: "pure CJK",
			text: "你好世界",
			want: 6,
		},
		{
			name: "mixed ASCII and CJK",
			text: "hello 你好",
			want: 4,
		},
		{
			name: "empty string",
			text: "",
			want: 0,
		},
		{
			// 100 ASCII chars at ~0.25 tokens each -> 25
			name: "longer ASCII text",
			text: strings.Repeat("a", 100),
			want: 25,
		},
		{
			name: "longer CJK text",
			text: strings.Repeat("测", 10),
			want: 15,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.text)
			if got != tt.want {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.text, got, tt.want)
			}
		})
	}
}

// A single CJK char is ~1.5 tokens; integer conversion floors to 1.
func TestEstimateTokens_SingleCJKChar(t *testing.T) {
	got := EstimateTokens("世")
	want := 1
	if got != want {
		t.Errorf("single CJK char: got %d, want %d", got, want)
	}
}

// CJK Extension A range (U+3400-U+4DBF) should be detected as CJK.
func TestEstimateTokens_CJKExtensionA(t *testing.T) {
	// U+3400 "㐀" is CJK Extension A, same weight as unified CJK -> 1 after flooring.
	got := EstimateTokens("㐀")
	want := 1
	if got != want {
		t.Errorf("CJK Extension A char: got %d, want %d", got, want)
	}
}

// Non-CJK Unicode (accented Latin) should NOT be counted as CJK.
func TestEstimateTokens_NonCJKUnicode(t *testing.T) {
	// "café" = 4 visible characters, no CJK -> heuristic approximation: 1
	text := "café"
	got := EstimateTokens(text)
	want := 1
	if got != want {
		t.Errorf("accented text: got %d, want %d", got, want)
	}
}

func TestCountTokensAPI_WhenAPIKeyIsMissing_ThenReturnsError(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	_, err := CountTokensAPI("hello")
	if err == nil {
		t.Fatal("CountTokensAPI returned nil error, want missing key error")
	}
	if !strings.Contains(err.Error(), "ANTHROPIC_API_KEY not set") {
		t.Fatalf("error = %v, want missing key", err)
	}
}

func TestCountTokens_SendsAnthropicRequestAndParsesResponse(t *testing.T) {
	var sawRequest bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRequest = true
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("x-api-key = %q, want test-key", got)
		}
		if got := r.Header.Get("anthropic-version"); got != apiVersion {
			t.Fatalf("anthropic-version = %q, want %q", got, apiVersion)
		}

		var payload struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Model != anthropicModel {
			t.Fatalf("model = %q, want %q", payload.Model, anthropicModel)
		}
		if len(payload.Messages) != 1 || payload.Messages[0].Role != "user" || payload.Messages[0].Content != "hello" {
			t.Fatalf("messages = %#v, want one user hello message", payload.Messages)
		}

		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"input_tokens":42}`))
	}))
	defer server.Close()

	got, err := countTokens("hello", "test-key", server.URL, server.Client())
	if err != nil {
		t.Fatalf("countTokens returned error: %v", err)
	}
	if !sawRequest {
		t.Fatal("server did not receive request")
	}
	if got != 42 {
		t.Fatalf("countTokens = %d, want 42", got)
	}
}

func TestCountTokens_WhenAPIReturnsError_ThenIncludesStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad key", http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := countTokens("hello", "test-key", server.URL, server.Client())
	if err == nil {
		t.Fatal("countTokens returned nil error, want API status error")
	}
	if !strings.Contains(err.Error(), "API returned status 401") {
		t.Fatalf("error = %v, want status 401", err)
	}
}
