## 1. Deterministic Filtering

- [ ] 1.1 Retain all risk/decision signals: errors, warnings, rollbacks, blockers, corrections, exit codes, git state, test summaries, final reports, and subagent blockers
- [ ] 1.2 Emit structured evidence summaries for every compressed or unknown tool call
- [ ] 1.3 Add tests proving important signals are retained and raw session files remain untouched

## 2. Redaction

- [ ] 2.1 Complete deny-list patterns and conservative high-entropy detection
- [ ] 2.2 Apply redaction to filtered artifact writes and evidence expansion reads by default
- [ ] 2.3 Add explicit config/flag-gated unredacted override, off by default
- [ ] 2.4 Add tests per secret category plus raw-session immutability checks

## 3. Evidence Store

- [ ] 3.1 Implement `manifest.json`, `normalized.jsonl`, `filtered.jsonl`, `evidence-index.json`, `handoff.json`, and `handoff.md` layout
- [ ] 3.2 Implement deterministic `evidence_id` derivation
- [ ] 3.3 Implement expand-by-ID with caller-specified size limit and truncation flag
- [ ] 3.4 Add path-traversal and symlink-escape protection
- [ ] 3.5 Add atomic writes and per-session advisory locking without holding locks across Local LLM calls
- [ ] 3.6 Add tests for stable IDs, traversal rejection, truncation, and concurrent writers

## 4. Handoff Validation

- [ ] 4.1 Validate/demote unevidenced claims
- [ ] 4.2 Reject or strip hallucinated evidence IDs with warnings
- [ ] 4.3 Flag deployment/rollback and branch/commit conflicts
- [ ] 4.4 Add mock OpenAI-compatible tests for repair failure, schema-invalid output, hallucinated evidence, and chunked merge correctness
