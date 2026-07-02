package distiller

import (
	"context"
	"os"
	"testing"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/config"
)

func TestLiveLocalLLMIntegration_OptIn(t *testing.T) {
	if os.Getenv("SESSION_CONTEXT_LIVE_LLM_TEST") != "1" {
		t.Skip("set SESSION_CONTEXT_LIVE_LLM_TEST=1 to run live Local LLM integration")
	}
	baseURL := os.Getenv("LOCAL_LLM_BASE_URL")
	model := os.Getenv("LOCAL_LLM_MODEL")
	if baseURL == "" || model == "" {
		t.Fatal("LOCAL_LLM_BASE_URL and LOCAL_LLM_MODEL are required for live test")
	}
	temp := 0.0
	cfg := config.LocalLLMConfig{
		Enabled:         true,
		BaseURL:         baseURL,
		APIKey:          os.Getenv("LOCAL_LLM_API_KEY"),
		Model:           model,
		MaxContext:      32000,
		MaxOutputTokens: 1000,
		TimeoutSeconds:  120,
		Temperature:     &temp,
	}
	_, diag, err := Generate(context.Background(), Request{
		Config:             cfg,
		Session:            testSession(),
		FilteredTranscript: "User asked to create a handoff. Tests were not run. Evidence evi-demo exists only as text.",
	}, NewClient(cfg))
	if err != nil {
		t.Fatalf("live Generate returned error: %v", err)
	}
	if diag.Model != model || diag.RedactedInputChars == 0 {
		t.Fatalf("unexpected diagnostics: %#v", diag)
	}
}
