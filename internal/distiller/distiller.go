package distiller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/config"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/handoff"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/redaction"
)

type Request struct {
	Config             config.LocalLLMConfig
	Session            handoff.SessionInfo
	FilteredTranscript string
}

type Diagnostics struct {
	Chunks             int
	Repaired           bool
	RedactedInputChars int
	RawOutputChars     int
	Model              string
}

type InvalidOutputError struct {
	Raw string
	Err error
}

func (e InvalidOutputError) Error() string {
	return fmt.Sprintf("local LLM output did not validate after repair: %v", e.Err)
}

func Generate(ctx context.Context, req Request, client Client) (handoff.Handoff, Diagnostics, error) {
	if !req.Config.IsEnabled() {
		return handoff.Handoff{}, Diagnostics{}, fmt.Errorf("local LLM is not enabled or is missing base_url/model")
	}
	if client.BaseURL == "" {
		client = NewClient(req.Config)
	}
	filtered := redaction.RedactSecrets(req.FilteredTranscript)
	chunks := ChunkTranscript(filtered, req.Config.MaxContext)
	diag := Diagnostics{
		Chunks:             len(chunks),
		RedactedInputChars: len(filtered),
		Model:              req.Config.Model,
	}
	var merged handoff.Handoff
	for i, chunk := range chunks {
		label := fmt.Sprintf("%d/%d", i+1, len(chunks))
		h, rawChars, repaired, err := generateChunk(ctx, req, client, chunk, label)
		diag.RawOutputChars += rawChars
		if repaired {
			diag.Repaired = true
		}
		if err != nil {
			return handoff.Handoff{}, diag, err
		}
		if i == 0 {
			merged = h
			continue
		}
		merged = mergeHandoffs(merged, h)
	}
	merged.Session = req.Session
	merged.Session.Model = req.Config.Model
	merged = handoff.NormalizeAndValidate(merged, nil)
	return merged, diag, nil
}

func generateChunk(ctx context.Context, req Request, client Client, chunk string, label string) (handoff.Handoff, int, bool, error) {
	raw, err := client.Chat(ctx, BuildHandoffMessages(req.Session, chunk, label), req.Config.MaxOutputTokens)
	if err != nil {
		return handoff.Handoff{}, 0, false, err
	}
	h, parseErr := parseHandoff(raw)
	if parseErr == nil {
		return h, len(raw), false, nil
	}
	repairedRaw, repairErr := client.Chat(ctx, BuildRepairMessages(raw, parseErr), req.Config.MaxOutputTokens)
	if repairErr != nil {
		return handoff.Handoff{}, len(raw), true, InvalidOutputError{Raw: raw, Err: repairErr}
	}
	h, parseErr = parseHandoff(repairedRaw)
	if parseErr != nil {
		return handoff.Handoff{}, len(raw) + len(repairedRaw), true, InvalidOutputError{Raw: repairedRaw, Err: parseErr}
	}
	return h, len(raw) + len(repairedRaw), true, nil
}

func parseHandoff(raw string) (handoff.Handoff, error) {
	candidate := strings.TrimSpace(raw)
	candidate = strings.TrimPrefix(candidate, "```json")
	candidate = strings.TrimPrefix(candidate, "```")
	candidate = strings.TrimSuffix(candidate, "```")
	candidate = strings.TrimSpace(candidate)
	if start := strings.Index(candidate, "{"); start > 0 {
		candidate = candidate[start:]
	}
	if end := strings.LastIndex(candidate, "}"); end >= 0 && end < len(candidate)-1 {
		candidate = candidate[:end+1]
	}
	var h handoff.Handoff
	if err := json.Unmarshal([]byte(candidate), &h); err != nil {
		return handoff.Handoff{}, err
	}
	return handoff.NormalizeAndValidate(h, nil), nil
}

func mergeHandoffs(a, b handoff.Handoff) handoff.Handoff {
	if a.Objective == "" {
		a.Objective = b.Objective
	}
	if a.ImplementationState.Summary == "" {
		a.ImplementationState.Summary = b.ImplementationState.Summary
	} else if b.ImplementationState.Summary != "" {
		a.ImplementationState.Summary += "\n" + b.ImplementationState.Summary
	}
	a.ConfirmedDecisions = append(a.ConfirmedDecisions, b.ConfirmedDecisions...)
	a.RejectedOrSuperseded = append(a.RejectedOrSuperseded, b.RejectedOrSuperseded...)
	a.Verification.Passed = append(a.Verification.Passed, b.Verification.Passed...)
	a.Verification.Failed = append(a.Verification.Failed, b.Verification.Failed...)
	a.Verification.NotRun = append(a.Verification.NotRun, b.Verification.NotRun...)
	a.Verification.Warnings = append(a.Verification.Warnings, b.Verification.Warnings...)
	a.KnownBlockers = append(a.KnownBlockers, b.KnownBlockers...)
	a.UnresolvedQuestions = append(a.UnresolvedQuestions, b.UnresolvedQuestions...)
	a.NextActions = append(a.NextActions, b.NextActions...)
	a.UserCorrections = append(a.UserCorrections, b.UserCorrections...)
	a.ClaimsRequiringReverification = append(a.ClaimsRequiringReverification, b.ClaimsRequiringReverification...)
	a.WorkflowImprovementCandidates = append(a.WorkflowImprovementCandidates, b.WorkflowImprovementCandidates...)
	a.Validation.Warnings = append(a.Validation.Warnings, b.Validation.Warnings...)
	a.Validation.Conflicts = append(a.Validation.Conflicts, b.Validation.Conflicts...)
	return a
}
