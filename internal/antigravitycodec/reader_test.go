package antigravitycodec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/analyzer"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

func TestCodec_ParseStandaloneShapeFixture(t *testing.T) {
	path := filepath.Join("testdata", "antigravity-standalone-redacted", ".system_generated", "logs", "transcript_full.jsonl")
	codec := Codec{}
	ref, err := codec.inspectRef(path)
	if err != nil {
		t.Fatalf("inspectRef returned error: %v", err)
	}
	if ref.ID != "antigravity-standalone-redacted" {
		t.Fatalf("ref.ID = %q", ref.ID)
	}
	if ref.ProjectPath != `D:\repo\example` {
		t.Fatalf("ProjectPath = %q", ref.ProjectPath)
	}
	if ref.FirstPrompt != "Please inspect this repo and run tests." {
		t.Fatalf("FirstPrompt = %q", ref.FirstPrompt)
	}

	events, parseErrors, err := codec.Parse(ref)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(parseErrors) != 0 {
		t.Fatalf("parseErrors = %#v, want none", parseErrors)
	}
	if len(events) != 8 {
		t.Fatalf("got %d events, want 8", len(events))
	}
	if events[0].EventType != "message" || events[0].Role != "user" || events[0].Content != "Please inspect this repo and run tests." {
		t.Fatalf("user event = %#v", events[0])
	}
	if events[2].Tool == nil || events[2].Tool.Name != "run_command" || !strings.Contains(events[2].Tool.Arguments, "git status") {
		t.Fatalf("tool call event = %#v", events[2])
	}
	if events[3].Tool == nil || events[3].Tool.Name != "run_command" || events[3].Tool.Status != "ok" {
		t.Fatalf("tool result event = %#v", events[3])
	}
	if events[7].Tool == nil || events[7].Tool.Name != "error_message" || events[7].Tool.Status != "FAILED" {
		t.Fatalf("error event = %#v", events[7])
	}
	if events[2].EventID == "" || events[2].EventID != session.StableEventID(ref.ID, session.ProviderAntigravity, events[2].Sequence, events[2].Source, events[2].EventType) {
		t.Fatalf("event id not stable: %#v", events[2])
	}
}

func TestCodec_ReadAllFeedsExistingAnalyzer(t *testing.T) {
	path := filepath.Join("testdata", "antigravity-standalone-redacted", ".system_generated", "logs", "transcript_full.jsonl")
	events, err := (Codec{}).ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	stats := analyzer.ComputeStats(events)
	if stats.RawChars <= stats.FilteredChars {
		t.Fatalf("expected filtered to be smaller: raw=%d filtered=%d", stats.RawChars, stats.FilteredChars)
	}
	if stats.Categories["user_text"] == 0 || stats.Categories["assistant_text"] == 0 {
		t.Fatalf("conversation text not retained: %#v", stats.Categories)
	}
	if stats.PerTool["run_command"] == nil || stats.PerTool["run_command"].CallCount != 1 {
		t.Fatalf("per-tool stats missing run_command: %#v", stats.PerTool)
	}
}

func TestCodec_ParseMalformedLineContinues(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, "agy-bad", ".system_generated", "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(logDir, "transcript_full.jsonl")
	data := strings.Join([]string{
		`{"step_index":0,"source":"USER_EXPLICIT","type":"USER_INPUT","status":"DONE","created_at":"2026-06-30T09:29:15Z","content":"<USER_REQUEST>start</USER_REQUEST>"}`,
		`{"step_index":1,"source":"MODEL","type":"PLANNER_RESPONSE",`,
		`{"step_index":2,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","created_at":"2026-06-30T09:29:16Z","content":"after"}`,
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	ref := session.SessionRef{ID: "agy-bad", Provider: session.ProviderAntigravity, Path: path}
	events, parseErrors, err := (Codec{}).Parse(ref)
	if err != nil {
		t.Fatalf("Parse returned fatal error: %v", err)
	}
	if len(parseErrors) != 1 {
		t.Fatalf("parseErrors = %#v, want one", parseErrors)
	}
	if len(events) != 2 || events[1].Content != "after" {
		t.Fatalf("events = %#v, want valid lines preserved", events)
	}
}

func TestCodec_DiscoverUsesEnvRoots(t *testing.T) {
	root := t.TempDir()
	sessionDir := filepath.Join(root, "standalone-session", ".system_generated", "logs")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	src := filepath.Join("testdata", "antigravity-standalone-redacted", ".system_generated", "logs", "transcript_full.jsonl")
	dst := filepath.Join(sessionDir, "transcript_full.jsonl")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write fixture copy: %v", err)
	}
	t.Setenv("ANTIGRAVITY_SESSION_ROOTS", root)

	refs, err := (Codec{}).Discover()
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if len(refs) != 1 || refs[0].ID != "standalone-session" {
		t.Fatalf("refs = %#v", refs)
	}
}

func TestStatusText_DoesNotTreatFileContentErrorWordsAsExecutionFailure(t *testing.T) {
	content := "File Path: `incident.md`\n1: # Error handling and failed attempts"
	if got := statusText("VIEW_FILE", "DONE", content); got != "ok" {
		t.Fatalf("statusText() = %q, want ok", got)
	}
	if got := statusText("MCP_TOOL", "DONE", "Encountered error in step execution: Error: Tool execution failed"); got != "FAILED" {
		t.Fatalf("statusText() = %q, want FAILED", got)
	}
}
