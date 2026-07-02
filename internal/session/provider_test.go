package session

import "testing"

func TestStableEventID_GivenSameSourceIdentity_ThenStable(t *testing.T) {
	source := EventSource{
		Path:      "C:/Users/peter/.codex/sessions/2026/07/02/session.jsonl",
		LineStart: 12,
		ByteStart: 4096,
	}

	got := StableEventID("session-1", ProviderCodex, 7, source, "tool_call")
	again := StableEventID("session-1", ProviderCodex, 7, source, "tool_call")
	if got != again {
		t.Fatalf("StableEventID was not stable: %q vs %q", got, again)
	}
	if len(got) != len("ev-0123456789abcdef") {
		t.Fatalf("StableEventID length = %d, want %d", len(got), len("ev-0123456789abcdef"))
	}
}

func TestStableEventID_GivenContentChangesOnly_ThenUnchanged(t *testing.T) {
	before := EventSource{
		Path:        "session.jsonl",
		LineStart:   2,
		ByteStart:   18,
		ContentHash: "sha256:before",
	}
	after := before
	after.ContentHash = "sha256:after"

	got := StableEventID("session-1", ProviderCodex, 2, before, "message")
	if changed := StableEventID("session-1", ProviderCodex, 2, after, "message"); got != changed {
		t.Fatalf("StableEventID changed when only content hash changed: %q vs %q", got, changed)
	}
}

func TestStableEventID_GivenDifferentSourceIdentity_ThenDifferent(t *testing.T) {
	source := EventSource{Path: "session.jsonl", LineStart: 1, ByteStart: 0}
	got := StableEventID("session-1", ProviderCodex, 1, source, "message")

	source.LineStart = 2
	if changed := StableEventID("session-1", ProviderCodex, 1, source, "message"); got == changed {
		t.Fatalf("StableEventID did not change after source identity changed: %q", got)
	}
}
