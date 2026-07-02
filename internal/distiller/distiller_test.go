package distiller

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/config"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/handoff"
)

func TestClient_GivenEmptyAPIKey_ThenOmitsAuthorizationHeader(t *testing.T) {
	var sawAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("authorization")
		writeChatResponse(t, w, minimalHandoffJSON())
	}))
	defer server.Close()

	cfg := localConfig(server.URL)
	cfg.APIKey = ""
	_, _, err := Generate(context.Background(), Request{
		Config:             cfg,
		Session:            testSession(),
		FilteredTranscript: "user asked for a handoff",
	}, NewClient(cfg))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if sawAuth != "" {
		t.Fatalf("authorization header = %q, want empty", sawAuth)
	}
}

func TestGenerate_GivenFilteredTranscriptWithSecret_ThenRedactsBeforeRequest(t *testing.T) {
	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requestBody = string(body)
		writeChatResponse(t, w, minimalHandoffJSON())
	}))
	defer server.Close()

	cfg := localConfig(server.URL)
	_, _, err := Generate(context.Background(), Request{
		Config:             cfg,
		Session:            testSession(),
		FilteredTranscript: `tool output included "api_key":"sk-test-redaction-token-1234567890"`,
	}, NewClient(cfg))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if strings.Contains(requestBody, "sk-test-redaction-token-1234567890") {
		t.Fatalf("secret leaked into request body: %s", requestBody)
	}
	if !strings.Contains(requestBody, "[REDACTED_SECRET]") && !strings.Contains(requestBody, "[REDACTED_TOKEN]") {
		t.Fatalf("request body missing redaction marker: %s", requestBody)
	}
	for _, want := range []string{`"temperature":0`, `"top_p":0.95`, `"top_k":20`, `"max_tokens":1000`} {
		if !strings.Contains(requestBody, want) {
			t.Fatalf("request body missing sampling field %s: %s", want, requestBody)
		}
	}
}

func TestGenerate_GivenInvalidJSONThenRepair_ThenUsesRepairedOutput(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			writeChatResponse(t, w, `not-json`)
			return
		}
		writeChatResponse(t, w, minimalHandoffJSON())
	}))
	defer server.Close()

	cfg := localConfig(server.URL)
	got, diag, err := Generate(context.Background(), Request{
		Config:             cfg,
		Session:            testSession(),
		FilteredTranscript: "work happened",
	}, NewClient(cfg))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if !diag.Repaired || calls != 2 {
		t.Fatalf("diag=%#v calls=%d, want repaired with two calls", diag, calls)
	}
	if got.Objective != "Resume the session" {
		t.Fatalf("Objective = %q", got.Objective)
	}
}

func TestGenerate_GivenOversizedTranscript_ThenChunksAndMerges(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeChatResponse(t, w, strings.ReplaceAll(minimalHandoffJSON(), "Resume the session", "Chunk result"))
	}))
	defer server.Close()

	cfg := localConfig(server.URL)
	cfg.MaxContext = 2
	_, diag, err := Generate(context.Background(), Request{
		Config:             cfg,
		Session:            testSession(),
		FilteredTranscript: strings.Repeat("line\n", 20),
	}, NewClient(cfg))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if diag.Chunks <= 1 || calls <= 1 {
		t.Fatalf("diag=%#v calls=%d, want multiple chunks", diag, calls)
	}
}

func localConfig(baseURL string) config.LocalLLMConfig {
	temperature := 0.0
	topP := 0.95
	return config.LocalLLMConfig{
		Enabled:         true,
		BaseURL:         baseURL,
		Model:           "mock-model",
		MaxContext:      32000,
		MaxOutputTokens: 1000,
		TimeoutSeconds:  5,
		Temperature:     &temperature,
		TopP:            &topP,
		TopK:            20,
	}
}

func testSession() handoff.SessionInfo {
	return handoff.SessionInfo{
		Provider:      "codex",
		SessionID:     "session-1",
		SourcePath:    "session.jsonl",
		Workspace:     "D:/repo/session-context-broker",
		RawChars:      100,
		FilteredChars: 50,
	}
}

func minimalHandoffJSON() string {
	return `{
  "schema_version":"session-context-handoff/v1",
  "session":{"provider":"codex","session_id":"session-1","source_path":"session.jsonl","workspace":"D:/repo","model":"mock-model","raw_chars":100,"filtered_chars":50},
  "objective":"Resume the session",
  "confirmed_decisions":[],
  "rejected_or_superseded":[],
  "implementation_state":{"summary":"implemented mock handoff","changed_files":[],"current_branch":"","current_commit":""},
  "verification":{"passed":[],"failed":[],"not_run":[],"warnings":[]},
  "deployment":{"completed":false,"environment":"","evidence_refs":[],"rollback":[]},
  "known_blockers":[],
  "unresolved_questions":[],
  "next_actions":[{"action":"Continue implementation","evidence_refs":[]}],
  "user_corrections":[],
  "claims_requiring_reverification":[],
  "workflow_improvement_candidates":[],
  "validation":{"warnings":[],"conflicts":[]}
}`
}

func writeChatResponse(t *testing.T, w http.ResponseWriter, content string) {
	t.Helper()
	w.Header().Set("content-type", "application/json")
	err := json.NewEncoder(w).Encode(map[string]any{
		"choices": []map[string]any{
			{"message": map[string]any{"content": content}},
		},
	})
	if err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
