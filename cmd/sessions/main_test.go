package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"claude-code-session-reader/internal/parser"
)

func TestSampleCount(t *testing.T) {
	tests := []struct {
		name      string
		requested int
		total     int
		want      int
	}{
		{
			name:      "negative request shows none",
			requested: -1,
			total:     3,
			want:      0,
		},
		{
			name:      "request larger than total is capped",
			requested: 10,
			total:     3,
			want:      3,
		},
		{
			name:      "request within total is unchanged",
			requested: 2,
			total:     3,
			want:      2,
		},
		{
			name:      "zero request shows none",
			requested: 0,
			total:     3,
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sampleCount(tt.requested, tt.total)
			if got != tt.want {
				t.Fatalf("sampleCount(%d, %d) = %d, want %d", tt.requested, tt.total, got, tt.want)
			}
		})
	}
}

func TestRunAudit_WhenSamplesIsNegative_ThenShowsZeroSamplesWithoutPanic(t *testing.T) {
	root := t.TempDir()
	sid := "12345678-1234-1234-1234-123456789abc"
	projectDir := filepath.Join(root, ".claude", "projects", "proj")
	metaDir := filepath.Join(root, ".claude", "usage-data", "session-meta")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("create project dir: %v", err)
	}
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatalf("create meta dir: %v", err)
	}

	transcript := `{"type":"user","timestamp":"2026-05-28T00:00:01Z","toolUseResult":{"success":true,"commandName":"Bash"},"message":{"role":"user","content":[{"type":"tool_result","content":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(projectDir, sid+".jsonl"), []byte(transcript), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	oldProjectsDir := parser.ProjectsDir
	oldSessionMetaDir := parser.SessionMetaDir
	parser.ProjectsDir = filepath.Join(root, ".claude", "projects")
	parser.SessionMetaDir = metaDir
	t.Cleanup(func() {
		parser.ProjectsDir = oldProjectsDir
		parser.SessionMetaDir = oldSessionMetaDir
	})

	var stdout, stderr bytes.Buffer
	err := runAudit([]string{sid, "-n", "-1"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runAudit returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "=== tool_result_cut (1 items, showing 0) ===") {
		t.Fatalf("stdout missing zero-sample header:\n%s", out)
	}
	if !strings.Contains(out, "... and 1 more") {
		t.Fatalf("stdout missing remaining-sample count:\n%s", out)
	}
}
