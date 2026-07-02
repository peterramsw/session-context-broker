// Package handoff defines the structured derived artifact produced from a
// filtered session transcript.
package handoff

import "encoding/json"

const (
	SchemaVersion = "session-context-handoff/v1"
	Disclosure    = "This handoff is a derived artifact and must not be treated as the source of truth."
)

type Handoff struct {
	SchemaVersion                 string              `json:"schema_version"`
	Session                       SessionInfo         `json:"session"`
	Objective                     string              `json:"objective"`
	ConfirmedDecisions            []EvidenceClaim     `json:"confirmed_decisions"`
	RejectedOrSuperseded          []EvidenceClaim     `json:"rejected_or_superseded"`
	ImplementationState           ImplementationState `json:"implementation_state"`
	Verification                  VerificationState   `json:"verification"`
	Deployment                    DeploymentState     `json:"deployment"`
	KnownBlockers                 []EvidenceClaim     `json:"known_blockers"`
	UnresolvedQuestions           []TextItem          `json:"unresolved_questions"`
	NextActions                   []NextAction        `json:"next_actions"`
	UserCorrections               []EvidenceClaim     `json:"user_corrections"`
	ClaimsRequiringReverification []EvidenceClaim     `json:"claims_requiring_reverification"`
	WorkflowImprovementCandidates []TextItem          `json:"workflow_improvement_candidates"`
	Validation                    ValidationReport    `json:"validation"`
}

type SessionInfo struct {
	Provider      string `json:"provider"`
	SessionID     string `json:"session_id"`
	SourcePath    string `json:"source_path"`
	Workspace     string `json:"workspace"`
	Model         string `json:"model"`
	RawChars      int    `json:"raw_chars"`
	FilteredChars int    `json:"filtered_chars"`
}

type EvidenceClaim struct {
	Claim        string   `json:"claim"`
	EvidenceRefs []string `json:"evidence_refs"`
}

type NextAction struct {
	Action       string   `json:"action"`
	EvidenceRefs []string `json:"evidence_refs"`
}

type TextItem string

func (t *TextItem) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*t = TextItem(text)
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	for _, key := range []string{"text", "claim", "candidate", "question", "warning", "conflict", "action", "summary"} {
		if value, ok := obj[key].(string); ok && value != "" {
			*t = TextItem(value)
			return nil
		}
	}
	encoded, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	*t = TextItem(string(encoded))
	return nil
}

func (c *EvidenceClaim) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		c.Claim = text
		c.EvidenceRefs = []string{}
		return nil
	}
	type alias EvidenceClaim
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*c = EvidenceClaim(decoded)
	if c.EvidenceRefs == nil {
		c.EvidenceRefs = []string{}
	}
	return nil
}

func (a *NextAction) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		a.Action = text
		a.EvidenceRefs = []string{}
		return nil
	}
	type alias NextAction
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*a = NextAction(decoded)
	if a.EvidenceRefs == nil {
		a.EvidenceRefs = []string{}
	}
	return nil
}

type ImplementationState struct {
	Summary       string   `json:"summary"`
	ChangedFiles  []string `json:"changed_files"`
	CurrentBranch string   `json:"current_branch"`
	CurrentCommit string   `json:"current_commit"`
}

type VerificationState struct {
	Passed   []EvidenceClaim `json:"passed"`
	Failed   []EvidenceClaim `json:"failed"`
	NotRun   []EvidenceClaim `json:"not_run"`
	Warnings []EvidenceClaim `json:"warnings"`
}

type DeploymentState struct {
	Completed    bool            `json:"completed"`
	Environment  string          `json:"environment"`
	EvidenceRefs []string        `json:"evidence_refs"`
	Rollback     []EvidenceClaim `json:"rollback"`
}

type ValidationReport struct {
	Warnings  []TextItem `json:"warnings"`
	Conflicts []TextItem `json:"conflicts"`
}
