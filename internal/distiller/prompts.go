package distiller

import (
	"fmt"
	"strings"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/handoff"
)

func BuildHandoffMessages(session handoff.SessionInfo, filteredTranscript string, chunkLabel string) []Message {
	system := strings.Join([]string{
		"You produce strict JSON for a session handoff.",
		"Use only the filtered transcript and metadata provided by the user message.",
		"Do not invent evidence IDs. If no evidence ID is present, leave evidence_refs empty and put uncertain claims under claims_requiring_reverification.",
		"Return only one JSON object. No Markdown fences. No commentary.",
	}, "\n")
	user := fmt.Sprintf(`Session metadata:
provider=%s
session_id=%s
workspace=%s
source_path=%s
raw_chars=%d
filtered_chars=%d
chunk=%s

Required JSON object shape:
{
  "schema_version": "%s",
  "session": {"provider":"","session_id":"","source_path":"","workspace":"","model":"","raw_chars":0,"filtered_chars":0},
  "objective": "",
  "confirmed_decisions": [{"claim":"","evidence_refs":[]}],
  "rejected_or_superseded": [{"claim":"","evidence_refs":[]}],
  "implementation_state": {"summary":"","changed_files":[],"current_branch":"","current_commit":""},
  "verification": {"passed":[],"failed":[],"not_run":[],"warnings":[]},
  "deployment": {"completed":false,"environment":"","evidence_refs":[],"rollback":[]},
  "known_blockers": [{"claim":"","evidence_refs":[]}],
  "unresolved_questions": [],
  "next_actions": [{"action":"","evidence_refs":[]}],
  "user_corrections": [{"claim":"","evidence_refs":[]}],
  "claims_requiring_reverification": [{"claim":"","evidence_refs":[]}],
  "workflow_improvement_candidates": [],
  "validation": {"warnings":[],"conflicts":[]}
}

Filtered transcript:
%s`, session.Provider, session.SessionID, session.Workspace, session.SourcePath, session.RawChars, session.FilteredChars, chunkLabel, handoff.SchemaVersion, filteredTranscript)
	return []Message{{Role: "system", Content: system}, {Role: "user", Content: user}}
}

func BuildRepairMessages(raw string, parseErr error) []Message {
	return []Message{
		{
			Role:    "system",
			Content: "Repair malformed JSON. Return only one valid JSON object matching the requested handoff shape. No Markdown fences.",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Parse error: %v\n\nMalformed output:\n%s", parseErr, raw),
		},
	}
}
