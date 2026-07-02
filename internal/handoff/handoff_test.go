package handoff

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeAndValidate_GivenUnknownEvidenceRef_ThenWarnsAndStrips(t *testing.T) {
	h := Handoff{
		ConfirmedDecisions: []EvidenceClaim{{Claim: "decided", EvidenceRefs: []string{"ev-good", "ev-bad"}}},
	}
	got := NormalizeAndValidate(h, map[string]bool{"ev-good": true})
	if len(got.ConfirmedDecisions[0].EvidenceRefs) != 1 || got.ConfirmedDecisions[0].EvidenceRefs[0] != "ev-good" {
		t.Fatalf("EvidenceRefs = %#v, want only ev-good", got.ConfirmedDecisions[0].EvidenceRefs)
	}
	if len(got.Validation.Warnings) != 1 || !strings.Contains(string(got.Validation.Warnings[0]), "ev-bad") {
		t.Fatalf("Warnings = %#v, want ev-bad warning", got.Validation.Warnings)
	}
}

func TestNormalizeAndValidate_GivenUnevidencedPassedTestClaim_ThenDemotes(t *testing.T) {
	h := Handoff{
		Verification: VerificationState{
			Passed: []EvidenceClaim{{Claim: "go test ./... passed"}},
		},
	}
	got := NormalizeAndValidate(h, map[string]bool{"evi-good": true})
	if len(got.Verification.Passed) != 0 {
		t.Fatalf("Verification.Passed = %#v, want claim demoted", got.Verification.Passed)
	}
	if len(got.ClaimsRequiringReverification) != 1 || got.ClaimsRequiringReverification[0].Claim != "go test ./... passed" {
		t.Fatalf("ClaimsRequiringReverification = %#v", got.ClaimsRequiringReverification)
	}
}

func TestEvidenceClaim_UnmarshalString_ThenNormalizesToClaim(t *testing.T) {
	var claims []EvidenceClaim
	if err := json.Unmarshal([]byte(`["warning text"]`), &claims); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if len(claims) != 1 || claims[0].Claim != "warning text" || len(claims[0].EvidenceRefs) != 0 {
		t.Fatalf("claims = %#v", claims)
	}
}

func TestNextAction_UnmarshalString_ThenNormalizesToAction(t *testing.T) {
	var actions []NextAction
	if err := json.Unmarshal([]byte(`["continue"]`), &actions); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if len(actions) != 1 || actions[0].Action != "continue" || len(actions[0].EvidenceRefs) != 0 {
		t.Fatalf("actions = %#v", actions)
	}
}

func TestTextItem_UnmarshalObject_ThenNormalizesToText(t *testing.T) {
	var items []TextItem
	if err := json.Unmarshal([]byte(`[{"candidate":"make sampling explicit"}]`), &items); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if len(items) != 1 || string(items[0]) != "make sampling explicit" {
		t.Fatalf("items = %#v", items)
	}
}

func TestNormalizeAndValidate_GivenDeploymentAndRollback_ThenConflict(t *testing.T) {
	h := Handoff{
		Deployment: DeploymentState{
			Completed: true,
			Rollback:  []EvidenceClaim{{Claim: "rolled back"}},
		},
	}
	got := NormalizeAndValidate(h, nil)
	if len(got.Validation.Conflicts) != 1 {
		t.Fatalf("Conflicts = %#v, want one", got.Validation.Conflicts)
	}
}

func TestRenderMarkdown_IncludesDisclosure(t *testing.T) {
	md := RenderMarkdown(Handoff{Session: SessionInfo{Provider: "codex", SessionID: "abc"}})
	if !strings.Contains(md, Disclosure) {
		t.Fatalf("markdown missing disclosure:\n%s", md)
	}
}

func TestWriteArtifacts_WritesJSONAndMarkdown(t *testing.T) {
	dir := t.TempDir()
	h := Handoff{SchemaVersion: SchemaVersion, Session: SessionInfo{Provider: "codex", SessionID: "abc"}}
	out, err := WriteArtifacts(dir, h, false)
	if err != nil {
		t.Fatalf("WriteArtifacts returned error: %v", err)
	}
	for _, name := range []string{"handoff.json", "handoff.md"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("%s not written: %v", name, err)
		}
	}
	if _, err := WriteArtifacts(dir, h, false); err == nil {
		t.Fatalf("WriteArtifacts without force overwrote existing output")
	}
}

func TestWriteFilteredArtifact_WritesRedactedFilteredTranscript(t *testing.T) {
	dir := t.TempDir()
	info := SessionInfo{
		Provider:      "codex",
		SessionID:     "abc",
		SourcePath:    "/tmp/session.jsonl",
		Workspace:     "/tmp/proj",
		RawChars:      100,
		FilteredChars: 25,
	}

	path, err := WriteFilteredArtifact(dir, info, "hello [REDACTED_SECRET]\n", false)
	if err != nil {
		t.Fatalf("WriteFilteredArtifact returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read filtered artifact: %v", err)
	}
	text := string(data)
	for _, want := range []string{"# Filtered Session", "Provider: codex", "Session: abc", "hello [REDACTED_SECRET]"} {
		if !strings.Contains(text, want) {
			t.Fatalf("filtered artifact missing %q:\n%s", want, text)
		}
	}
	if _, err := WriteFilteredArtifact(dir, info, "again\n", false); err == nil {
		t.Fatalf("WriteFilteredArtifact without force overwrote existing output")
	}
}
