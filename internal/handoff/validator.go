package handoff

import "fmt"

func NormalizeAndValidate(h Handoff, evidence map[string]bool) Handoff {
	if h.SchemaVersion == "" {
		h.SchemaVersion = SchemaVersion
	}
	h.ConfirmedDecisions = cleanClaimsWithPolicy(h.ConfirmedDecisions, "confirmed_decisions", evidence, &h, true)
	h.RejectedOrSuperseded = cleanClaims(h.RejectedOrSuperseded, "rejected_or_superseded", evidence, &h)
	h.KnownBlockers = cleanClaimsWithPolicy(h.KnownBlockers, "known_blockers", evidence, &h, true)
	h.UserCorrections = cleanClaimsWithPolicy(h.UserCorrections, "user_corrections", evidence, &h, true)
	h.Verification.Passed = cleanClaimsWithPolicy(h.Verification.Passed, "verification.passed", evidence, &h, true)
	h.Verification.Failed = cleanClaimsWithPolicy(h.Verification.Failed, "verification.failed", evidence, &h, true)
	h.Verification.Warnings = cleanClaimsWithPolicy(h.Verification.Warnings, "verification.warnings", evidence, &h, true)
	h.Verification.NotRun = cleanClaims(h.Verification.NotRun, "verification.not_run", evidence, &h)
	h.Deployment.EvidenceRefs = cleanEvidenceRefs("deployment.evidence_refs", h.Deployment.EvidenceRefs, evidence, &h)
	for i := range h.Deployment.Rollback {
		h.Deployment.Rollback[i].EvidenceRefs = cleanEvidenceRefs("deployment.rollback", h.Deployment.Rollback[i].EvidenceRefs, evidence, &h)
	}
	for i := range h.NextActions {
		h.NextActions[i].EvidenceRefs = cleanEvidenceRefs("next_actions", h.NextActions[i].EvidenceRefs, evidence, &h)
	}
	if h.Deployment.Completed && len(h.Deployment.Rollback) > 0 {
		h.Validation.Conflicts = append(h.Validation.Conflicts, TextItem("deployment.completed is true while rollback evidence is present"))
	}
	if h.ImplementationState.CurrentBranch != "" || h.ImplementationState.CurrentCommit != "" {
		h.Validation.Warnings = append(h.Validation.Warnings, TextItem("implementation branch/commit state requires verify_workspace re-verification"))
	}
	h.ensureNonNilSlices()
	return h
}

func (h *Handoff) ensureNonNilSlices() {
	h.ConfirmedDecisions = nonNilClaims(h.ConfirmedDecisions)
	h.RejectedOrSuperseded = nonNilClaims(h.RejectedOrSuperseded)
	h.KnownBlockers = nonNilClaims(h.KnownBlockers)
	h.UnresolvedQuestions = nonNilTextItems(h.UnresolvedQuestions)
	h.NextActions = nonNilNextActions(h.NextActions)
	h.UserCorrections = nonNilClaims(h.UserCorrections)
	h.ClaimsRequiringReverification = nonNilClaims(h.ClaimsRequiringReverification)
	h.WorkflowImprovementCandidates = nonNilTextItems(h.WorkflowImprovementCandidates)
	h.Verification.Passed = nonNilClaims(h.Verification.Passed)
	h.Verification.Failed = nonNilClaims(h.Verification.Failed)
	h.Verification.NotRun = nonNilClaims(h.Verification.NotRun)
	h.Verification.Warnings = nonNilClaims(h.Verification.Warnings)
	h.Deployment.EvidenceRefs = nonNilStrings(h.Deployment.EvidenceRefs)
	h.Deployment.Rollback = nonNilClaims(h.Deployment.Rollback)
	h.Validation.Warnings = nonNilTextItems(h.Validation.Warnings)
	h.Validation.Conflicts = nonNilTextItems(h.Validation.Conflicts)
	for i := range h.ConfirmedDecisions {
		h.ConfirmedDecisions[i].EvidenceRefs = nonNilStrings(h.ConfirmedDecisions[i].EvidenceRefs)
	}
	for i := range h.NextActions {
		h.NextActions[i].EvidenceRefs = nonNilStrings(h.NextActions[i].EvidenceRefs)
	}
}

func nonNilClaims(v []EvidenceClaim) []EvidenceClaim {
	if v == nil {
		return []EvidenceClaim{}
	}
	return v
}

func nonNilNextActions(v []NextAction) []NextAction {
	if v == nil {
		return []NextAction{}
	}
	return v
}

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func nonNilTextItems(v []TextItem) []TextItem {
	if v == nil {
		return []TextItem{}
	}
	return v
}

func cleanClaims(claims []EvidenceClaim, field string, evidence map[string]bool, h *Handoff) []EvidenceClaim {
	return cleanClaimsWithPolicy(claims, field, evidence, h, false)
}

func cleanClaimsWithPolicy(claims []EvidenceClaim, field string, evidence map[string]bool, h *Handoff, requireEvidence bool) []EvidenceClaim {
	out := make([]EvidenceClaim, 0, len(claims))
	for _, claim := range claims {
		claim.EvidenceRefs = cleanEvidenceRefs(field, claim.EvidenceRefs, evidence, h)
		if requireEvidence && evidence != nil && len(claim.EvidenceRefs) == 0 && claim.Claim != "" {
			h.ClaimsRequiringReverification = append(h.ClaimsRequiringReverification, claim)
			h.Validation.Warnings = append(h.Validation.Warnings, TextItem(fmt.Sprintf("a %s claim was moved to claims_requiring_reverification (no resolvable evidence)", field)))
			continue
		}
		out = append(out, claim)
	}
	return out
}

func cleanEvidenceRefs(field string, refs []string, evidence map[string]bool, h *Handoff) []string {
	if evidence == nil {
		return refs
	}
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if evidence[ref] {
			out = append(out, ref)
		} else {
			h.Validation.Warnings = append(h.Validation.Warnings, TextItem(fmt.Sprintf("%s referenced unknown evidence id %q", field, ref)))
		}
	}
	return out
}
