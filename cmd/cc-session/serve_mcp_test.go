package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/parser"
)

func TestRunServeMCP_GivenToolsList_ThenAdvertisesSessionTools(t *testing.T) {
	configPath := writeMCPConfigFile(t, t.TempDir(), map[string]any{"storage_root": t.TempDir()})
	input := framedJSON(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"})
	var stdout, stderr bytes.Buffer
	err := runServeMCP([]string{"--config", configPath}, strings.NewReader(input), &stdout, &stderr, parser.Store{}, testReader)
	if err != nil {
		t.Fatalf("runServeMCP returned error: %v stderr=%s", err, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{"list_sessions", "create_handoff", "expand_evidence", "verify_workspace"} {
		if !strings.Contains(got, want) {
			t.Fatalf("MCP tools/list missing %s in %s", want, got)
		}
	}
}

func TestRunServeMCP_GivenThreeClients_ThenReadToolsCoexist(t *testing.T) {
	configPath := writeMCPConfigFile(t, t.TempDir(), map[string]any{"storage_root": t.TempDir()})
	var wg sync.WaitGroup
	errs := make(chan error, 3)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			input := framedJSON(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"})
			var stdout, stderr bytes.Buffer
			err := runServeMCP([]string{"--config", configPath}, strings.NewReader(input), &stdout, &stderr, parser.Store{}, testReader)
			if err != nil {
				errs <- err
				return
			}
			if !strings.Contains(stdout.String(), "list_sessions") {
				errs <- os.ErrInvalid
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent serve-mcp returned error: %v", err)
		}
	}
}

func writeMCPConfigFile(t *testing.T, dir string, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
