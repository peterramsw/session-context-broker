## 1. Deterministic Filtering

- [x] 1.1 Retain all risk/decision signals: errors, warnings, rollbacks, blockers, corrections, exit codes, git state, test summaries, final reports, and subagent blockers
- [x] 1.2 Emit structured evidence summaries for every compressed or unknown tool call
- [x] 1.3 Add tests proving important signals are retained and raw session files remain untouched

## 2. Redaction

- [x] 2.1 Complete deny-list patterns and conservative high-entropy detection
- [x] 2.2 Apply redaction to filtered artifact writes and evidence expansion reads by default
- [x] 2.3 Add explicit config/flag-gated unredacted override, off by default
- [x] 2.4 Add tests per secret category plus raw-session immutability checks

## 3. Evidence Store

- [x] 3.1 Implement `manifest.json`, `normalized.jsonl`, `filtered.jsonl`, `evidence-index.json`, `handoff.json`, and `handoff.md` layout
- [x] 3.2 Implement deterministic `evidence_id` derivation
- [x] 3.3 Implement expand-by-ID with caller-specified size limit and truncation flag
- [x] 3.4 Add path-traversal and symlink-escape protection
- [x] 3.5 Add atomic writes and per-session advisory locking without holding locks across Local LLM calls
- [x] 3.6 Add tests for stable IDs, traversal rejection, truncation, and concurrent writers

## 4. Handoff Validation

- [x] 4.1 Validate/demote unevidenced claims
- [x] 4.2 Reject or strip hallucinated evidence IDs with warnings
- [x] 4.3 Flag deployment/rollback and branch/commit conflicts
- [x] 4.4 Add mock OpenAI-compatible tests for repair failure, schema-invalid output, hallucinated evidence, and chunked merge correctness

## 5. Evidence grounding gap (found in live LLM testing)

- [x] 5.1 Feed the evidence index into the Local LLM distiller input — fixed: `renderEvidenceList` builds a bounded evidence listing (evidence_id + summary), `distiller.Request.EvidenceList` carries it, and `BuildHandoffMessages` instructs the model to cite only those IDs. Verified end-to-end against GB10: the same session that previously produced 0 confirmed decisions now yields 13 confirmed decisions and 20 claims with real `evidence_refs`. The demotion guard (4.1) still strips any hallucinated IDs.
- [x] 5.2 Fix demotion-warning wording — fixed: reworded to `"a <section> claim was moved to claims_requiring_reverification (no resolvable evidence)"`, which reads as a demotion audit entry rather than a phantom defect on an empty section
