# Design: Evidence, Filtering & Secret-Redaction Safety Layer

## Decision

Build the safety layer as four separable pieces — deterministic filtering,
evidence store, secret redaction, handoff validation — so each is testable
and reasonable in isolation, and so the layer works fully even when the
optional Local LLM is off. This is deliberate: `filtered-first` (no LLM)
must never depend on anything in this change that requires the LLM to be
enabled, and everything the LLM path adds on top must degrade to
"demoted, not fabricated" when evidence is missing.

## Architecture

```
raw session
  → deterministic filter        (internal/claudecodec, internal/analyzer)
      retains: errors, warnings, rollbacks, blockers, corrections,
      exit codes, git state, test summaries, subagent blockers
  → secret redaction             (internal/redaction)
      applied at filtered-output write AND at evidence-expansion read
  → evidence store                (internal/evidence)
      manifest.json / normalized.jsonl / filtered.jsonl /
      evidence-index.json / handoff.json / handoff.md
      under storage_root/<provider>/<session-id>/
  → [optional] Local LLM distillation
  → handoff validation            (internal/handoff)
      unevidenced claims demoted to claims_requiring_reverification;
      hallucinated evidence IDs stripped with a warning;
      deployment-completed + rollback-evidence-present → conflict
```

## Evidence store

- Evidence IDs are derived deterministically from event identity + source
  position (`StableEvidenceID`), so re-processing an unmodified session
  yields the same IDs — required for evidence_refs in a handoff to stay
  resolvable across runs.
- Expansion re-reads the original byte range by ID, enforces a caller size
  limit, reports truncation, and rejects path traversal / symlink escape
  outside configured roots.
- Writes are atomic (temp file + rename) with a short-lived per-session
  advisory lock held only around the write, never across a Local LLM call —
  so one process waiting on a slow LLM response never blocks unrelated
  reads from another process against the same store.

## Redaction

Applied at two boundaries only: filtered-artifact writes and evidence
expansion reads. Raw session files on disk are never touched — redaction is
a read/write-boundary concern, not a mutation of source data. Unredacted
access requires an explicit, off-by-default config/flag override.

## Handoff validation

`NormalizeAndValidate` demotes any claim lacking a resolvable evidence ID to
`claims_requiring_reverification` rather than letting it stand as
confirmed, and flags a small set of direct conflicts (deployment completed
while rollback evidence exists; branch/commit state that needs
`verify_workspace` re-verification).

## Evidence grounding gap (found during live GB10 testing, since fixed)

The pieces above were each individually correct but did not compose: the
Local LLM distiller was never given the evidence index, only the filtered
transcript, so every claim's `evidence_refs` came back empty and got
demoted — technically correct behavior, but it meant the LLM path never
actually produced a confirmed decision. Fixed by rendering a bounded
evidence listing (`evidence_id` + summary) into the distiller prompt and
instructing the model to cite only IDs present in that listing. Verified
against the real GB10 endpoint: the same session that previously produced
zero confirmed decisions now produces confirmed decisions with real,
resolvable `evidence_refs`. The demotion guard still strips any ID the
model invents anyway.

## Consequences

- This change deliberately does not add MCP tools or Skills — it is the
  core those surfaces call, built and tested first.
- The evidence-grounding fix means the distiller prompt (and therefore
  input token cost per Local LLM call) grows with evidence-index size; a
  bounded/truncated listing keeps this predictable rather than unlimited.
