## 1. Architecture inventory & real-format investigation

- [x] 1.1 Write up the existing architecture inventory (module map, CLI entry, package responsibilities) as a draft for `docs/architecture.md` — reuse the Phase 1 baseline findings already gathered
- [x] 1.2 Investigate the real local Codex CLI session storage format/location on this machine (path, file format, metadata/turn/tool-call/tool-result/reasoning/compaction shapes)
- [x] 1.3 If a real local Codex session exists, redact and convert at least one sample into a test fixture; if none exists, build a fixture clearly labeled `synthetic` in its name/docs
- [x] 1.4 Record findings (schema/version targeted, real-vs-synthetic fixture status) as a draft for `docs/session-provider.md`

## 2. Normalized session schema & SessionProvider interface

- [x] 2.1 Define `SessionEvent`/`SessionMetadata`/`SessionRef` types and the `SessionProvider` interface in `internal/session/provider.go`
- [x] 2.2 Implement deterministic `event_id` derivation (hash of session_id/provider/sequence/source location/event_type)
- [ ] 2.3 Implement session discovery: config file entries → env var overrides (`CLAUDE_SESSION_ROOTS`, `CODEX_SESSION_ROOTS`) → Windows/Linux platform defaults, with an actionable error when nothing is found

## 3. Claude Code adapter compatibility

- [ ] 3.1 Wrap `internal/claudecodec` behind `SessionProvider` without changing its parsing behavior
- [x] 3.2 Run the full existing upstream test suite and confirm zero regressions
- [ ] 3.3 Add tests: malformed line, interrupted session, Unicode/Chinese content, unknown tool, large tool result, source-location round-trip (event → re-read original bytes)

## 4. Codex adapter

- [x] 4.1 Implement `internal/codexcodec` `Discover`/`Inspect`/`Parse` against the format investigated in Task 1.2
- [ ] 4.2 Handle user/assistant turns, tool call/result, reasoning events, and compaction events
- [x] 4.3 Implement graceful degradation (`event_type: "unknown"`, content preserved) for unrecognized Codex event shapes
- [x] 4.4 Add tests: real-or-synthetic fixture parse, malformed event, unknown event, interrupted session

## 5. Deterministic filtering extension

- [ ] 5.1 Extend `internal/summarizer`/`internal/analyzer` to guarantee retention of all listed risk/decision signals (errors, warnings, rollbacks, blockers, corrections, exit codes, git state, test summaries, final reports, subagent blockers)
- [ ] 5.2 Ensure every compressed or unknown-tool call emits a structured evidence summary (`evidence_id`, `event_id`, `tool_name`, `arguments_summary`, `result_summary`, `status`, `exit_code`, `warning_count`, `error_count`, `raw_chars`, `filtered_chars`, `source_location`) — never a bare status string
- [ ] 5.3 Add tests: prompt/reply retention, error/warning/rollback never disappear, exit codes retained, unknown-tool summaries present, large stdout remains expandable, raw session file untouched after filtering

## 6. Secret redaction

- [ ] 6.1 Implement `internal/redaction` with deny-list patterns (API keys, bearer/access/refresh tokens, passwords, cookies, client secrets, private keys, `.env` values, GitHub tokens, Anthropic/OpenAI-shaped tokens, JWTs, PEM blocks) plus a conservative high-entropy heuristic
- [ ] 6.2 Wire redaction into the filtered-transcript write path and the evidence-expand read path, on by default
- [ ] 6.3 Add an explicit config/flag-gated unredacted override (off by default)
- [ ] 6.4 Add tests per secret category, plus raw-session-immutability and no-third-party-network-egress checks

## 7. Evidence store

- [ ] 7.1 Implement `internal/evidence`: `manifest.json`/`normalized.jsonl`/`filtered.jsonl`/`evidence-index.json`/`handoff.json`/`handoff.md` layout under `storage_root`
- [ ] 7.2 Implement deterministic `evidence_id` derivation
- [ ] 7.3 Implement expand-by-ID with a caller-specified size limit and a truncation flag in the response
- [ ] 7.4 Implement path-traversal and symlink-escape protection for both evidence-store paths and session-root reads
- [ ] 7.5 Implement concurrency safety: atomic temp-file-then-rename writes for every artifact, plus a short-lived per-session advisory lock (e.g. `github.com/gofrs/flock`) only around the create-handoff critical section — never held across the local LLM HTTP call
- [ ] 7.6 Add tests: stable evidence IDs across reruns, truncation flag correctness, traversal/symlink rejection, concurrent multi-process create-handoff test (no corruption, no deadlock), proof that a slow/stuck distiller call in one process doesn't block another process's unrelated reads

## 8. Handoff schema & validator

- [x] 8.1 Define the handoff Go structs + JSON schema in `internal/handoff/schema.go` (evidence: `artifacts/local-llm-smoke-test-2026-07-02.md`)
- [ ] 8.2 Implement `internal/handoff/validator.go`: evidence-ref existence checks, unevidenced-claim demotion to `claims_requiring_reverification`, deployment/rollback conflict detection, branch/commit mismatch warning
- [x] 8.3 Implement `internal/handoff/renderer.go` producing `handoff.md`, always including the required derived-artifact disclosure sentence verbatim (evidence: `artifacts/local-llm-smoke-test-2026-07-02.md`)
- [ ] 8.4 Add tests: hallucinated evidence ID rejected, deployment/rollback conflict flagged, unevidenced-claim warning generated, disclosure sentence present in every render

## 9. Local LLM distiller

- [x] 9.1 Implement `internal/distiller/openai_client.go`: configurable base_url/api_key/model/timeout (no hardcoding), empty-API-key support for unauthenticated local endpoints (evidence: `artifacts/local-llm-smoke-test-2026-07-02.md`)
- [ ] 9.2 Implement `internal/distiller/prompts.go`: input restricted to filtered transcript + evidence metadata + schema + extraction rules, with an explicit debug-mode gate before raw session content is ever included
- [ ] 9.3 Implement `internal/distiller/chunker.go`: phase-heuristic segmentation (requirement/planning/design/implementation/debugging/testing/deployment/rollback/final-report) preserving event order when the transcript exceeds `local_llm.max_context`
- [ ] 9.4 Implement `internal/distiller/merger.go`: merge per-chunk partial results, dropping (with a warning) any evidence ref not present in the evidence index
- [x] 9.5 Implement the one-repair-then-fail-loud flow, preserving the raw failed LLM output on disk when repair doesn't recover a valid schema (evidence: `artifacts/local-llm-smoke-test-2026-07-02.md`)
- [x] 9.6 Check whether existing `internal/tokens` is sufficient for the new token-estimate needs before adding any new tokenizer dependency (evidence: `artifacts/local-llm-smoke-test-2026-07-02.md`)
- [ ] 9.7 Add tests against a mock OpenAI-compatible server: valid JSON, invalid JSON, successful repair, failed repair, timeout, endpoint-unavailable, schema-invalid output, hallucinated evidence ID, chunked distillation + merge correctness, deployment/rollback conflict, passed-test-missing-evidence warning

## 10. CLI commands

- [ ] 10.1 Add `list --provider {claude_code|codex|all}`, `inspect`, `filter`, `handoff [--force] [--provider]`, `search`, `expand <evidence-id>`, `verify-workspace --workspace <path>`, `serve-mcp` subcommands to `cmd/cc-session`
- [ ] 10.2 Ensure every new subcommand calls shared `internal/*` packages only — no logic duplicated between CLI and MCP
- [x] 10.3 Re-run the full existing test suite to confirm `list`/`read`/`context`/`stats`/`audit`/`expand`/`inject` are unaffected

## 11. MCP stdio server

- [ ] 11.1 Evaluate and pick a Go MCP SDK: attempt `modelcontextprotocol/go-sdk` first (build + minimal stdio smoke test), fall back to `mark3labs/mcp-go` if it proves unworkable; record the decision in `docs/mcp-tools.md`
- [ ] 11.2 Implement `internal/mcp/server.go` + `tools.go` wiring all nine tools to the shared core packages
- [ ] 11.3 Implement `verify_workspace` as a fixed set of read-only git plumbing/porcelain calls only, enforcing `allowed_workspace_roots`
- [ ] 11.4 Implement/verify multi-process concurrency: smoke test with three concurrent `serve-mcp` processes against the same evidence store
- [ ] 11.5 Add tests: all tool schemas, session-not-found, provider-not-found, path traversal, symlink escape, workspace-root restriction, evidence size limit, local-LLM-unavailable behavior, malformed handoff handling, stdio server smoke test, three-process concurrency test

## 12. Skills

- [ ] 12.1 Write `skills/common/resume-session.md`, `skills/common/close-session.md`, `skills/common/review-history.md` implementing the Resume/Close/Review-History workflows exactly as specified (Resume calls `verify_workspace` and never trusts unverified completion claims; Review History only proposes candidates, never auto-edits rule files)
- [ ] 12.2 Write thin `skills/claude-code/SKILL.md`, `skills/codex/SKILL.md`, and `skills/antigravity/SKILL.md` wrappers that reference the shared content, matching each platform's actual skill-installation convention (Claude Code: `allowed-tools` frontmatter, installs to `~/.claude/skills/`; Antigravity 2.0: `tools` frontmatter referencing MCP tool names, installs to `~/.gemini/skills/`; Codex: confirm its actual convention during this task rather than assuming)
- [ ] 12.3 Document Antigravity 2.0's MCP config (`~/.gemini/config/mcp_config.json`) and Skill install path in `docs/skills.md` and `docs/mcp-tools.md`, verified against this machine's actual installed Antigravity version
- [ ] 12.4 Manually verify the Claude Code skill installs (to `~/.claude/skills/`) and activates correctly; manually verify the Antigravity skill installs (to `~/.gemini/skills/`) and is selectable
- [ ] 12.5 Update install scripts to offer install-time client selection for Claude Code, Codex, and Antigravity, showing already-installed targets as checked/selected by default, plus a non-interactive `--clients {all|none|claude,codex,antigravity}` path; normal setup must not require a separate `cc-session init`

## 13. Config & documentation

- [ ] 13.1 Implement config loading: `SESSION_CONTEXT_CONFIG` env var if set, else default to `<storage_root>/config.json` (i.e. `~/.session-context/config.json` by default) — note a real local config already exists at that path on this machine (GB10 Bifrost/local LLM endpoint) and must load correctly once this task is done; plus all other specified env var overrides, matching the `session_sources`/`storage_root`/`allowed_workspace_roots`/`local_llm.*` schema
- [ ] 13.2 Write `docs/architecture.md`, `docs/session-provider.md`, `docs/normalized-event-schema.md`, `docs/handoff-schema.md`, `docs/local-llm-distillation.md`, `docs/mcp-tools.md` (including Claude Code, Codex, and Antigravity MCP config examples — check Antigravity's real local config format rather than guessing), `docs/skills.md`, `docs/security.md`, `docs/upstream-sync.md`
- [ ] 13.3 Update `README.md`: fork attribution line, preserved-feature list, Codex support, local LLM configuration, CLI usage, MCP install (all three clients), Skill install (Claude Code + Codex), Resume/Close workflow walkthroughs, storage location, redaction behavior, known limitations, upstream-sync method

## 14. End-to-end validation & final report

- [ ] 14.1 Run a full pipeline e2e test: fixture session → discover → inspect → parse → normalize → filter → redact → mock/local LLM distill → validate → write handoff → MCP `get_handoff` → `search_session` → `expand_evidence` → `verify_workspace`
- [ ] 14.2 Add an opt-in live local LLM integration test that is skipped by default in `go test ./...`
- [x] 14.3 Re-run real-session smoke tests (Claude Code, already partly validated in Phase 1) and attempt a real-or-synthetic Codex smoke test (evidence: `artifacts/provider-smoke-test-2026-07-02.md`, `artifacts/filtered-first-policy-smoke-test-2026-07-02.md`)
- [x] 14.4 Confirm all upstream + new tests pass and `go build ./...` is clean (evidence: `artifacts/provider-smoke-test-2026-07-02.md`, `artifacts/filtered-first-policy-smoke-test-2026-07-02.md`)
- [ ] 14.5 Compile the final report in the 16-section format the user specified, backing every claim with the actual command and its output
