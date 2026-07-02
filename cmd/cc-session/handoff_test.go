package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/parser"
)

func TestRunHandoff_GivenClaudeSessionAndMockLocalLLM_ThenWritesArtifacts(t *testing.T) {
	root, sid := writeCLIFixture(t)
	storageRoot := t.TempDir()
	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := new(bytes.Buffer)
		_, _ = body.ReadFrom(r.Body)
		requestBody = body.String()
		writeMockHandoffResponse(t, w)
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.json")
	configJSON := fmt.Sprintf(`{
  "storage_root": %q,
  "local_llm": {
    "enabled": true,
    "base_url": %q,
    "api_key": "",
    "model": "mock-model",
    "max_context": 32000,
    "max_output_tokens": 1000,
    "timeout_seconds": 5,
    "temperature": 0,
    "top_p": 0.95,
    "top_k": 20
  }
}`, storageRoot, server.URL)
	if err := os.WriteFile(configPath, []byte(configJSON), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var stdout, stderr bytes.Buffer
	store := parser.Store{
		ProjectsDir:    filepath.Join(root, ".claude", "projects"),
		SessionMetaDir: filepath.Join(root, ".claude", "usage-data", "session-meta"),
	}
	err := runHandoff([]string{"--provider", "claude_code", "--config", configPath, "--llm", "always", "--force", sid}, &stdout, &stderr, store, testReader)
	if err != nil {
		t.Fatalf("runHandoff returned error: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Mode: llm") || !strings.Contains(stdout.String(), "Model: mock-model") || !strings.Contains(stdout.String(), "Output:") {
		t.Fatalf("stdout missing handoff summary:\n%s", stdout.String())
	}
	outDir := filepath.Join(storageRoot, "claude_code", sid)
	for _, name := range []string{"filtered.md", "handoff.json", "handoff.md"} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Fatalf("%s was not written: %v", name, err)
		}
	}
	if !strings.Contains(requestBody, `"temperature":0`) || !strings.Contains(requestBody, `"top_k":20`) {
		t.Fatalf("request body missing deterministic sampling config: %s", requestBody)
	}
}

func TestRunHandoff_GivenSmallSessionAutoAndNoLocalLLMConfig_ThenWritesFilteredOnly(t *testing.T) {
	root, sid := writeCLIFixture(t)
	storageRoot := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"storage_root":"`+filepath.ToSlash(storageRoot)+`"}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var stdout, stderr bytes.Buffer
	store := parser.Store{
		ProjectsDir:    filepath.Join(root, ".claude", "projects"),
		SessionMetaDir: filepath.Join(root, ".claude", "usage-data", "session-meta"),
	}
	err := runHandoff([]string{"--provider", "claude_code", "--config", configPath, sid}, &stdout, &stderr, store, testReader)
	if err != nil {
		t.Fatalf("runHandoff returned error: %v\nstderr=%s", err, stderr.String())
	}
	got := stdout.String()
	if !strings.Contains(got, "Mode: filtered") || !strings.Contains(got, "below threshold") {
		t.Fatalf("stdout missing filtered-only decision:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(storageRoot, "claude_code", sid, "filtered.md")); err != nil {
		t.Fatalf("filtered.md was not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(storageRoot, "claude_code", sid, "handoff.json")); err == nil {
		t.Fatalf("handoff.json was written even though Local LLM was skipped")
	}
}

func TestRunHandoff_GivenAlwaysLLMAndNoLocalLLMConfig_ThenErrorsAfterFilteredWrite(t *testing.T) {
	root, sid := writeCLIFixture(t)
	storageRoot := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{"storage_root":"`+filepath.ToSlash(storageRoot)+`"}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var stdout, stderr bytes.Buffer
	store := parser.Store{
		ProjectsDir:    filepath.Join(root, ".claude", "projects"),
		SessionMetaDir: filepath.Join(root, ".claude", "usage-data", "session-meta"),
	}
	err := runHandoff([]string{"--provider", "claude_code", "--config", configPath, "--llm", "always", sid}, &stdout, &stderr, store, testReader)
	if err == nil || !strings.Contains(err.Error(), "Local LLM is not enabled") {
		t.Fatalf("error = %v, want Local LLM config error", err)
	}
	if _, err := os.Stat(filepath.Join(storageRoot, "claude_code", sid, "filtered.md")); err != nil {
		t.Fatalf("filtered.md was not written before Local LLM error: %v", err)
	}
}

func TestRunHandoff_GivenAntigravitySessionAndLLMNever_ThenWritesFiltered(t *testing.T) {
	storageRoot := t.TempDir()
	t.Setenv("ANTIGRAVITY_SESSION_ROOTS", filepath.Join("..", "..", "internal", "antigravitycodec", "testdata"))

	var stdout, stderr bytes.Buffer
	err := runHandoff(
		[]string{"--provider", "antigravity", "--llm", "never", "--out", storageRoot, "--force", "antigravity-standalone"},
		&stdout,
		&stderr,
		parser.Store{},
		testReader,
	)
	if err != nil {
		t.Fatalf("runHandoff returned error: %v\nstderr=%s", err, stderr.String())
	}
	got := stdout.String()
	if !strings.Contains(got, "Provider: antigravity") || !strings.Contains(got, "Mode: filtered") {
		t.Fatalf("stdout missing Antigravity filtered summary:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(storageRoot, "antigravity", "antigravity-standalone-redacted", "filtered.md")); err != nil {
		t.Fatalf("filtered.md was not written: %v", err)
	}
}

func writeMockHandoffResponse(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	content := `{
  "schema_version":"session-context-handoff/v1",
  "session":{"provider":"claude_code","session_id":"session-1","source_path":"session.jsonl","workspace":"proj","model":"mock-model","raw_chars":100,"filtered_chars":50},
  "objective":"Resume prior work",
  "confirmed_decisions":[],
  "rejected_or_superseded":[],
  "implementation_state":{"summary":"Generated by mock Local LLM","changed_files":[],"current_branch":"","current_commit":""},
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
	w.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"choices": []map[string]any{{"message": map[string]any{"content": content}}},
	}); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
