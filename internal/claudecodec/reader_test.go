package claudecodec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

func TestParseLine_UserStringContent(t *testing.T) {
	event := parseLine(t, `{"type":"user","timestamp":"2026-05-28T00:00:00Z","message":{"role":"user","content":"hello"}}`)
	if event.Kind != session.EventUserMessage {
		t.Fatalf("kind = %s, want %s", event.Kind, session.EventUserMessage)
	}
	if event.User == nil || event.User.Text != "hello" {
		t.Fatalf("user text = %#v, want hello", event.User)
	}
}

func TestParseLine_UserUnknownContentShapeKeepsRawJSON(t *testing.T) {
	event := parseLine(t, `{"type":"user","timestamp":"2026-05-28T00:00:00Z","message":{"role":"user","content":{"unexpected":"shape"}}}`)
	if event.Kind != session.EventUserMessage {
		t.Fatalf("kind = %s, want %s", event.Kind, session.EventUserMessage)
	}
	if event.User == nil || event.User.Text != `{"unexpected":"shape"}` {
		t.Fatalf("user text = %#v, want raw JSON", event.User)
	}
}

func TestParseLine_AssistantBlocks(t *testing.T) {
	event := parseLine(t, `{"type":"assistant","timestamp":"2026-05-28T00:00:01Z","message":{"role":"assistant","content":[{"type":"text","text":"hi"},{"type":"thinking","thinking":"private reasoning"},{"type":"tool_use","name":"Bash","id":"tool-1","input":{"command":"echo ok","description":"Echo ok"}}]}}`)
	if event.Kind != session.EventAssistantMessage {
		t.Fatalf("kind = %s, want %s", event.Kind, session.EventAssistantMessage)
	}
	if event.Assistant == nil {
		t.Fatal("assistant is nil")
	}
	if event.Assistant.Text != "hi" {
		t.Fatalf("assistant text = %q, want hi", event.Assistant.Text)
	}
	if got := event.Assistant.Thinking; len(got) != 1 || got[0] != "private reasoning" {
		t.Fatalf("thinking = %#v", got)
	}
	if got := event.Assistant.ToolUses; len(got) != 1 || got[0].Name != "Bash" || got[0].ID != "tool-1" || got[0].Input.String("description") != "Echo ok" {
		t.Fatalf("tool uses = %#v", got)
	}
}

func TestParseLine_ToolResultStringContent(t *testing.T) {
	event := parseLine(t, `{"type":"user","timestamp":"2026-05-28T00:00:02Z","toolUseResult":{"success":true,"commandName":"Bash"},"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool-1","content":"ok"}]}}`)
	if event.Kind != session.EventToolResult {
		t.Fatalf("kind = %s, want %s", event.Kind, session.EventToolResult)
	}
	if event.Tool == nil || event.Tool.ToolUseID != "tool-1" || event.Tool.Text != "ok" || event.Tool.RawName != "Bash" || !event.Tool.Success {
		t.Fatalf("tool result = %#v", event.Tool)
	}
}

func TestParseLine_ToolResultTextBlockContent(t *testing.T) {
	event := parseLine(t, `{"type":"user","toolUseResult":{"success":false,"commandName":"Read"},"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool-2","content":[{"type":"text","text":"first part"},{"type":"text","text":"second part"}]}]}}`)
	if event.Tool == nil || event.Tool.Text != "first part\nsecond part" || event.Tool.Success {
		t.Fatalf("tool result = %#v", event.Tool)
	}
}

func TestParseLine_UserAnswer(t *testing.T) {
	event := parseLine(t, `{"type":"user","toolUseResult":{"success":true},"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool-3","content":"User has answered your questions: ship it"}]}}`)
	if event.User == nil || !event.User.IsAnswer || event.User.Text != "User has answered your questions: ship it" {
		t.Fatalf("user answer = %#v", event.User)
	}
}

func TestParseLine_Noise(t *testing.T) {
	event := parseLine(t, `{"type":"system","message":{"content":"system details"}}`)
	if event.Kind != session.EventNoise {
		t.Fatalf("kind = %s, want %s", event.Kind, session.EventNoise)
	}
	if event.Noise == nil || event.Noise.Text != "system details" {
		t.Fatalf("noise = %#v", event.Noise)
	}
}

func TestParseLine_UnknownEntryWithoutMessageIsSkipped(t *testing.T) {
	_, ok, err := ParseLine([]byte(`{"type":"future-event","payload":{"value":1}}`))
	if err != nil {
		t.Fatalf("ParseLine returned error: %v", err)
	}
	if ok {
		t.Fatal("unknown entry without message should be skipped")
	}
}

func TestReadFile_WhenLineIsMalformed_ThenReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.jsonl")
	data := strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":"ok"}}`,
		`{"type":"user","message":`,
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	err := ReadFile(path, func(session.Event) error { return nil })
	if err == nil {
		t.Fatal("ReadFile returned nil, want malformed JSON error")
	}
	if !strings.Contains(err.Error(), "parse transcript line") {
		t.Fatalf("error = %v, want parse transcript line", err)
	}
}

func TestReadFile_WhenLastLineHasNoTrailingNewline_ThenReadsEvent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	data := `{"type":"user","message":{"role":"user","content":"ok"}}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var events []session.Event
	err := ReadFile(path, func(event session.Event) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if len(events) != 1 || events[0].User == nil || events[0].User.Text != "ok" {
		t.Fatalf("events = %#v, want one user event", events)
	}
}

func TestCollectAgentToolIDs(t *testing.T) {
	events := []session.Event{
		{
			Kind: session.EventAssistantMessage,
			Assistant: &session.AssistantMessage{ToolUses: []session.ToolUse{
				{ID: "agent-1", Name: "Agent"},
				{ID: "bash-1", Name: "Bash"},
			}},
		},
	}
	ids := CollectAgentToolIDs(events)
	if !ids["agent-1"] || ids["bash-1"] {
		t.Fatalf("ids = %#v", ids)
	}
}

func TestParseLine_AllNoiseTypes(t *testing.T) {
	noiseTypes := []string{
		"file-history-snapshot", "attachment", "bridge-session",
		"last-prompt", "permission-mode", "mode", "ai-title",
		"custom-title", "agent-name", "pr-link",
		"queue-operation", "progress", "system",
	}
	for _, typ := range noiseTypes {
		t.Run(typ, func(t *testing.T) {
			line := fmt.Sprintf(`{"type":"%s","message":{"role":"user","content":"x"}}`, typ)
			event, ok, err := ParseLine([]byte(line))
			if err != nil {
				t.Fatalf("ParseLine(%s) error: %v", typ, err)
			}
			if !ok {
				t.Fatalf("ParseLine(%s) returned ok=false, want ok=true with noise event", typ)
			}
			if event.Kind != session.EventNoise {
				t.Fatalf("ParseLine(%s) kind = %s, want %s", typ, event.Kind, session.EventNoise)
			}
		})
	}
}

// ReadAll aggregates a whole transcript into an ordered event slice. This is
// the core entry point used by every CLI command, but until now it was only
// exercised indirectly via the out-of-process e2e tests (which don't count
// toward coverage). This test pins the aggregation directly: mixed entry types
// must each map to the right Kind, preserve file order, and carry their nested
// payloads (tool_use list, tool_result fields, noise text) intact.
func TestReadAll_GivenMixedEntryTypes_ThenAggregatesEventsInOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	// Order matters: user msg -> assistant w/ tool_use -> tool_result -> noise.
	// One blank line and one skippable-empty assistant are interleaved to prove
	// they don't shift the kept events' positions.
	lines := []string{
		`{"type":"user","timestamp":"2026-05-28T00:00:00Z","message":{"role":"user","content":"hello"}}`,
		`{"type":"assistant","timestamp":"2026-05-28T00:00:01Z","message":{"role":"assistant","content":[{"type":"text","text":"on it"},{"type":"tool_use","name":"Agent","id":"toolu_agent_1","input":{"prompt":"go"}}]}}`,
		`{"type":"user","timestamp":"2026-05-28T00:00:02Z","toolUseResult":{"success":true,"commandName":"Agent"},"message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_agent_1","content":"done"}]}}`,
		`{"type":"system","timestamp":"2026-05-28T00:00:03Z","message":{"role":"user","content":"system chatter"}}`,
		"",
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	events, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}

	// Four kept events: the empty trailing line is skipped, not turned into an event.
	wantKinds := []session.EventKind{
		session.EventUserMessage,
		session.EventAssistantMessage,
		session.EventToolResult,
		session.EventNoise,
	}
	if len(events) != len(wantKinds) {
		t.Fatalf("got %d events, want %d:\n%#v", len(events), len(wantKinds), events)
	}
	for i, want := range wantKinds {
		if events[i].Kind != want {
			t.Fatalf("event[%d].Kind = %s, want %s", i, events[i].Kind, want)
		}
	}

	// User message text preserved.
	if events[0].User == nil || events[0].User.Text != "hello" {
		t.Fatalf("event[0] user = %#v, want text 'hello'", events[0].User)
	}

	// Assistant text + nested tool_use aggregated into the same event.
	assistant := events[1].Assistant
	if assistant == nil || assistant.Text != "on it" {
		t.Fatalf("event[1] assistant = %#v, want text 'on it'", assistant)
	}
	if len(assistant.ToolUses) != 1 || assistant.ToolUses[0].Name != session.ToolAgent ||
		assistant.ToolUses[0].ID != "toolu_agent_1" {
		t.Fatalf("event[1] tool_use = %#v, want one Agent tool 'toolu_agent_1'", assistant.ToolUses)
	}

	// tool_result correlates back to the tool_use_id and carries success/text.
	tool := events[2].Tool
	if tool == nil || tool.ToolUseID != "toolu_agent_1" || tool.Text != "done" ||
		tool.RawName != "Agent" || !tool.Success {
		t.Fatalf("event[2] tool result = %#v", tool)
	}

	// Noise entry text extracted.
	if events[3].Noise == nil || events[3].Noise.Text != "system chatter" {
		t.Fatalf("event[3] noise = %#v, want text 'system chatter'", events[3].Noise)
	}

	// Agent tool IDs are collected from the aggregated events.
	agentIDs := CollectAgentToolIDs(events)
	if !agentIDs["toolu_agent_1"] {
		t.Fatalf("agent IDs = %#v, want toolu_agent_1 collected", agentIDs)
	}
	if len(agentIDs) != 1 {
		t.Fatalf("agent IDs = %#v, want exactly one entry", agentIDs)
	}
}

func TestReadAll_GivenEmptyFile_ThenReturnsNoEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.jsonl")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	events, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("got %d events, want 0:\n%#v", len(events), events)
	}
}

// A malformed line anywhere in the file aborts the whole read: ReadAll surfaces
// the parse error rather than silently returning a truncated event slice, so
// callers never operate on a partial transcript believing it complete.
func TestReadAll_GivenMalformedLineAmongValidLines_ThenReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mixed.jsonl")
	lines := []string{
		`{"type":"user","message":{"role":"user","content":"good line"}}`,
		`{"type":"user","message":`, // truncated JSON
		`{"type":"user","message":{"role":"user","content":"after the bad line"}}`,
		"",
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := ReadAll(path)
	if err == nil {
		t.Fatal("ReadAll returned nil, want parse error from the malformed line")
	}
	if !strings.Contains(err.Error(), "parse transcript line") {
		t.Fatalf("error = %v, want parse transcript line", err)
	}
}

// A noise entry's text is aggregated by extractAllText, which pulls from three
// sources that real transcripts populate: top-level text blocks, the nested
// blocks loop (thinking/tool_use/tool_result), and the toolUseResult CLI fields
// (stdout/stderr/output/content). This pins all three so the audit/stats noise
// surface keeps reflecting the full entry rather than dropping nested content.
func TestParseLine_NoiseEntry_ExtractsTextFromBlocksAndToolUseResult(t *testing.T) {
	line := `{"type":"system","timestamp":"2026-05-28T00:00:00Z",` +
		`"toolUseResult":{"stdout":"build ok","stderr":"warn: deprecated"},` +
		`"message":{"role":"user","content":[` +
		`{"type":"text","text":"running build"},` +
		`{"type":"tool_use","name":"Bash","id":"t1","input":{"command":"make"}},` +
		`{"type":"thinking","thinking":"deciding"}` +
		`]}}`
	event := parseLine(t, line)

	if event.Kind != session.EventNoise || event.Noise == nil {
		t.Fatalf("kind = %s, noise = %#v, want noise event", event.Kind, event.Noise)
	}

	// Order is deterministic: Message.Text() first, then the blocks loop
	// (thinking + tool_use input JSON), then toolUseResult fields in declared
	// key order (stdout, stderr, output, content). tool_result text blocks are
	// joined by Text() so the plain text block surfaces once.
	want := "running build\n" +
		`{"command":"make"}` + "\n" +
		"deciding\n" +
		"build ok\n" +
		"warn: deprecated"
	if event.Noise.Text != want {
		t.Fatalf("noise text =\n%q\nwant\n%q", event.Noise.Text, want)
	}
}

func parseLine(t *testing.T, line string) session.Event {
	t.Helper()
	event, ok, err := ParseLine([]byte(line))
	if err != nil {
		t.Fatalf("ParseLine returned error: %v", err)
	}
	if !ok {
		t.Fatal("ParseLine skipped line")
	}
	return event
}
