package codexcodec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/analyzer"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

func TestCodec_ParseRealShapeFixture(t *testing.T) {
	path := filepath.Join("testdata", "codex-real-shape-redacted.jsonl")
	codec := Codec{}
	ref, err := codec.inspectRef(path)
	if err != nil {
		t.Fatalf("inspectRef returned error: %v", err)
	}
	if ref.ID != "019f0000-0000-7000-8000-000000000001" {
		t.Fatalf("ref.ID = %q", ref.ID)
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
	if events[2].EventType != "message" || events[2].Role != "user" || events[2].Content != "measure this repo" {
		t.Fatalf("user event = %#v", events[2])
	}
	if events[4].Tool == nil || events[4].Tool.Name != "shell_command" || !strings.Contains(events[4].Tool.Arguments, "git status") {
		t.Fatalf("tool call event = %#v", events[4])
	}
	if events[5].Tool == nil || events[5].Tool.Status != "ok" || !strings.Contains(events[5].Tool.Result, "## main") {
		t.Fatalf("tool result event = %#v", events[5])
	}
	if events[7].EventType != "unknown" || !strings.Contains(events[7].Content, "preserve me") {
		t.Fatalf("unknown event = %#v", events[7])
	}
	if events[4].EventID == "" || events[4].EventID != session.StableEventID(ref.ID, session.ProviderCodex, events[4].Sequence, events[4].Source, events[4].EventType) {
		t.Fatalf("event id not stable: %#v", events[4])
	}
}

func TestCodec_ReadAllFeedsExistingAnalyzer(t *testing.T) {
	path := filepath.Join("testdata", "codex-real-shape-redacted.jsonl")
	codec := Codec{}
	events, err := codec.ReadAll(path)
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
	if stats.PerTool["shell_command"] == nil || stats.PerTool["shell_command"].CallCount != 1 {
		t.Fatalf("per-tool stats missing shell_command: %#v", stats.PerTool)
	}
}

func TestCodec_ParseMalformedLineContinues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout-test.jsonl")
	data := strings.Join([]string{
		`{"type":"session_meta","payload":{"id":"codex-bad","cwd":"D:\\repo\\example"}}`,
		`{"type":"response_item","payload":`,
		`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"after"}]}}`,
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	codec := Codec{}
	ref := session.SessionRef{ID: "codex-bad", Provider: session.ProviderCodex, Path: path}
	events, parseErrors, err := codec.Parse(ref)
	if err != nil {
		t.Fatalf("Parse returned fatal error: %v", err)
	}
	if len(parseErrors) != 1 {
		t.Fatalf("parseErrors = %#v, want one", parseErrors)
	}
	if len(events) != 2 {
		t.Fatalf("events = %#v, want valid lines preserved", events)
	}
	if events[1].Content != "after" {
		t.Fatalf("valid line after malformed one was lost: %#v", events)
	}
}

func TestCodec_ParseInterruptedSessionKeepsAvailableEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout-interrupted.jsonl")
	data := strings.Join([]string{
		`{"type":"session_meta","payload":{"id":"codex-interrupted","cwd":"D:\\repo\\example"}}`,
		`{"type":"event_msg","payload":{"type":"user_message","message":"start work"}}`,
		`{"type":"response_item","payload":{"type":"function_call","call_id":"call-1","name":"shell_command","arguments":{"command":"go test ./..."}}}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	codec := Codec{}
	ref := session.SessionRef{ID: "codex-interrupted", Provider: session.ProviderCodex, Path: path}
	events, parseErrors, err := codec.Parse(ref)
	if err != nil {
		t.Fatalf("Parse returned fatal error: %v", err)
	}
	if len(parseErrors) != 0 {
		t.Fatalf("parseErrors = %#v, want none", parseErrors)
	}
	if len(events) != 3 {
		t.Fatalf("events = %#v, want all available lines preserved", events)
	}
	if events[2].Tool == nil || events[2].Tool.CallID != "call-1" {
		t.Fatalf("interrupted tool call not preserved: %#v", events[2])
	}
}

func TestCodec_DiscoverUsesEnvRoots(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	src := filepath.Join("testdata", "codex-real-shape-redacted.jsonl")
	dst := filepath.Join(sessionsDir, "rollout-test.jsonl")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write fixture copy: %v", err)
	}
	t.Setenv("CODEX_SESSION_ROOTS", sessionsDir)

	refs, err := (Codec{}).Discover()
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if len(refs) != 1 || refs[0].ID != "019f0000-0000-7000-8000-000000000001" {
		t.Fatalf("refs = %#v", refs)
	}
}
