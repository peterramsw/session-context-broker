package evidence

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/handoff"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

func TestBuildIndex_GivenSameEvents_ThenStableEvidenceIDs(t *testing.T) {
	info, events := fixtureEvents(t)
	first := BuildIndex(info, events)
	second := BuildIndex(info, events)
	if len(first.Entries) == 0 {
		t.Fatal("BuildIndex returned no entries")
	}
	if first.Entries[0].EvidenceID != second.Entries[0].EvidenceID {
		t.Fatalf("EvidenceID changed: %s vs %s", first.Entries[0].EvidenceID, second.Entries[0].EvidenceID)
	}
}

func TestStoreExpand_GivenSecret_ThenRedactsByDefaultAndRawSourceUntouched(t *testing.T) {
	info, events := fixtureEvents(t)
	root := t.TempDir()
	wr, err := Store{Root: root}.Write(WriteInput{Session: info, Events: events, FilteredText: "filtered", Force: true})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	got, err := Store{Root: root}.Expand(ExpandOptions{
		Provider:     info.Provider,
		SessionID:    info.SessionID,
		EvidenceID:   wr.Index.Entries[0].EvidenceID,
		AllowedRoots: []string{filepath.Dir(info.SourcePath)},
		Limit:        4096,
	})
	if err != nil {
		t.Fatalf("Expand returned error: %v", err)
	}
	if strings.Contains(got.Content, "sk-test-redaction-token-1234567890") {
		t.Fatalf("expanded content leaked secret: %s", got.Content)
	}
	raw, _ := os.ReadFile(info.SourcePath)
	if !strings.Contains(string(raw), "sk-test-redaction-token-1234567890") {
		t.Fatalf("raw source was mutated: %s", string(raw))
	}
}

func TestStoreExpand_GivenTraversalOutsideAllowedRoots_ThenRejects(t *testing.T) {
	info, events := fixtureEvents(t)
	root := t.TempDir()
	wr, err := Store{Root: root}.Write(WriteInput{Session: info, Events: events, FilteredText: "filtered", Force: true})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	_, err = Store{Root: root}.Expand(ExpandOptions{
		Provider:     info.Provider,
		SessionID:    info.SessionID,
		EvidenceID:   wr.Index.Entries[0].EvidenceID,
		AllowedRoots: []string{t.TempDir()},
	})
	if err == nil {
		t.Fatal("Expand returned nil error for outside allowed roots")
	}
}

func TestStoreExpand_GivenLimit_ThenReportsTruncated(t *testing.T) {
	info, events := fixtureEvents(t)
	root := t.TempDir()
	wr, err := Store{Root: root}.Write(WriteInput{Session: info, Events: events, FilteredText: "filtered", Force: true})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	got, err := Store{Root: root}.Expand(ExpandOptions{
		Provider:     info.Provider,
		SessionID:    info.SessionID,
		EvidenceID:   wr.Index.Entries[0].EvidenceID,
		AllowedRoots: []string{filepath.Dir(info.SourcePath)},
		Limit:        10,
		Unredacted:   true,
	})
	if err != nil {
		t.Fatalf("Expand returned error: %v", err)
	}
	if !got.Truncated || got.BytesRead != 10 {
		t.Fatalf("Truncated=%v BytesRead=%d, want truncated 10 bytes", got.Truncated, got.BytesRead)
	}
}

func TestStoreWrite_GivenConcurrentWriters_ThenArtifactsRemainReadable(t *testing.T) {
	info, events := fixtureEvents(t)
	root := t.TempDir()
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = Store{Root: root}.Write(WriteInput{Session: info, Events: events, FilteredText: "filtered", Force: true})
		}()
	}
	wg.Wait()
	if _, err := (Store{Root: root}).ReadIndex(info.Provider, info.SessionID); err != nil {
		t.Fatalf("ReadIndex after concurrent writers returned error: %v", err)
	}
}

func fixtureEvents(t *testing.T) (handoff.SessionInfo, []session.SessionEvent) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	line := `{"message":"token sk-test-redaction-token-1234567890 and test failed"}`
	if err := os.WriteFile(path, []byte(line+"\n"), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}
	source := session.EventSource{Path: path, LineStart: 1, LineEnd: 1, ByteStart: 0, ByteEnd: int64(len(line) + 1)}
	event := session.SessionEvent{
		EventID:   session.StableEventID("session-1", session.ProviderCodex, 1, source, "tool_result"),
		SessionID: "session-1",
		Provider:  session.ProviderCodex,
		Sequence:  1,
		EventType: "tool_result",
		Content:   line,
		Tool:      &session.SessionTool{CallID: "call-1", Name: "Bash", Result: line, Status: "FAILED", HasExitCode: true, ExitCode: 1},
		Source:    source,
	}
	info := handoff.SessionInfo{Provider: session.ProviderCodex, SessionID: "session-1", SourcePath: path, RawChars: len(line), FilteredChars: 8}
	return info, []session.SessionEvent{event}
}
