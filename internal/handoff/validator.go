package handoff

import "fmt"

func NormalizeAndValidate(h Handoff, evidence map[string]bool) Handoff {
	if h.SchemaVersion == "" {
		h.SchemaVersion = SchemaVersion
	}
	h.ConfirmedDecisions = cleanClaims(h.ConfirmedDecisions, "confirmed_decisions", evidence, &h)
	h.RejectedOrSuperseded = cleanClaims(h.RejectedOrSuperseded, "rejected_or_superseded", evidence, &h)
	h.KnownBlockers = cleanClaims(h.KnownBlockers, "known_blockers", evidence, &h)
	h.UserCorrections = cleanClaims(h.UserCorrections, "user_corrections", evidence, &h)
	h.Verification.Passed = cleanClaims(h.Verification.Passed, "verification.passed", evidence, &h)
	h.Verification.Failed = cleanClaims(h.Verification.Failed, "verification.failed", evidence, &h)
	h.Verification.Warnings = cleanClaims(h.Verification.Warnings, "verification.warnings", evidence, &h)
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
	out := make([]EvidenceClaim, 0, len(claims))
	for _, claim := range claims {
		claim.EvidenceRefs = cleanEvidenceRefs(field, claim.EvidenceRefs, evidence, h)
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
