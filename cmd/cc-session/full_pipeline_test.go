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

func TestFullPipeline_GivenClaudeFixture_ThenWritesEvidenceAndHandoff(t *testing.T) {
	root, sid := writeCLIFixture(t)
	storageRoot := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeMockHandoffResponse(t, w)
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.json")
	temp := 0.0
	data, _ := json.Marshal(map[string]any{
		"storage_root":            storageRoot,
		"session_sources":         map[string]any{"claude_code": map[string]any{"roots": []string{filepath.Join(root, ".claude", "projects")}}},
		"allowed_workspace_roots": []string{root},
		"local_llm": map[string]any{
			"enabled":           true,
			"base_url":          server.URL,
			"model":             "mock-model",
			"max_context":       32000,
			"max_output_tokens": 1000,
			"temperature":       temp,
		},
	})
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var stdout, stderr bytes.Buffer
	store := parser.Store{
		ProjectsDir:    filepath.Join(root, ".claude", "projects"),
		SessionMetaDir: filepath.Join(root, ".claude", "usage-data", "session-meta"),
	}
	err := runHandoff([]string{"--provider", "claude_code", "--config", configPath, "--llm", "always", "--force", sid}, &stdout, &stderr, store, testReader)
	if err != nil {
		t.Fatalf("runHandoff returned error: %v stderr=%s", err, stderr.String())
	}
	for _, name := range []string{"manifest.json", "normalized.jsonl", "filtered.jsonl", "filtered.md", "evidence-index.json", "handoff.json", "handoff.md"} {
		if _, err := os.Stat(filepath.Join(storageRoot, "claude_code", sid, name)); err != nil {
			t.Fatalf("%s missing from pipeline output: %v\nstdout=%s", name, err, stdout.String())
		}
	}
	if !strings.Contains(stdout.String(), "Evidence index:") {
		t.Fatalf("stdout missing evidence index:\n%s", stdout.String())
	}
}

func TestVerifyWorkspace_GivenOutsideRoot_ThenRefuses(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	data := fmt.Sprintf(`{"storage_root":%q,"allowed_workspace_roots":[%q]}`, t.TempDir(), t.TempDir())
	if err := os.WriteFile(configPath, []byte(data), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var stdout, stderr bytes.Buffer
	err := runVerifyWorkspace([]string{"--config", configPath, t.TempDir()}, &stdout, &stderr, parser.Store{}, testReader)
	if err == nil {
		t.Fatalf("runVerifyWorkspace returned nil error for outside root, stdout=%s", stdout.String())
	}
}
