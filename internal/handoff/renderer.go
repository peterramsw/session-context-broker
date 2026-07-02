package handoff

import (
	"fmt"
	"strings"
)

func RenderMarkdown(h Handoff) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Session Handoff\n\n%s\n\n", Disclosure)
	fmt.Fprintf(&b, "Provider: %s\n\nSession: %s\n\n", h.Session.Provider, h.Session.SessionID)
	if h.Objective != "" {
		fmt.Fprintf(&b, "## Objective\n\n%s\n\n", h.Objective)
	}
	if h.ImplementationState.Summary != "" {
		fmt.Fprintf(&b, "## Implementation State\n\n%s\n\n", h.ImplementationState.Summary)
	}
	writeClaims(&b, "Confirmed Decisions", h.ConfirmedDecisions)
	writeClaims(&b, "Known Blockers", h.KnownBlockers)
	writeClaims(&b, "Verification Passed", h.Verification.Passed)
	writeClaims(&b, "Verification Failed", h.Verification.Failed)
	writeClaims(&b, "Claims Requiring Reverification", h.ClaimsRequiringReverification)
	if len(h.NextActions) > 0 {
		b.WriteString("## Next Actions\n\n")
		for _, action := range h.NextActions {
			fmt.Fprintf(&b, "- %s", action.Action)
			if len(action.EvidenceRefs) > 0 {
				fmt.Fprintf(&b, " `%s`", strings.Join(action.EvidenceRefs, "`, `"))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if len(h.Validation.Warnings) > 0 || len(h.Validation.Conflicts) > 0 {
		b.WriteString("## Validation\n\n")
		for _, warning := range h.Validation.Warnings {
			fmt.Fprintf(&b, "- Warning: %s\n", string(warning))
		}
		for _, conflict := range h.Validation.Conflicts {
			fmt.Fprintf(&b, "- Conflict: %s\n", string(conflict))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func writeClaims(b *strings.Builder, title string, claims []EvidenceClaim) {
	if len(claims) == 0 {
		return
	}
	fmt.Fprintf(b, "## %s\n\n", title)
	for _, claim := range claims {
		fmt.Fprintf(b, "- %s", claim.Claim)
		if len(claim.EvidenceRefs) > 0 {
			fmt.Fprintf(b, " `%s`", strings.Join(claim.EvidenceRefs, "`, `"))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
}
