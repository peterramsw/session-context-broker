## Why

`cc-session-reader` already saves tokens by filtering Claude Code transcripts outside the LLM context window. This phase extends that proven value without turning the tool into a heavyweight platform: it adds Codex and Google Antigravity standalone app session support, keeps deterministic filtered output as the source of truth, and adds an optional filtered-first Local LLM handoff artifact for sessions large enough to benefit from it.

The scope is intentionally narrow. Users with no Local LLM must still be able to list, inspect, filter, stats-check, and create filtered handoff artifacts for supported providers. Local LLM output is a derived acceleration layer, not the source of truth.

## What Changes

- Add normalized provider support for Codex CLI sessions and Google Antigravity standalone app sessions alongside existing Claude Code behavior.
- Add Antigravity standalone app parsing for `~/.gemini/antigravity/brain/<conversation-id>/.system_generated/logs/{transcript_full.jsonl,transcript.jsonl}`, explicitly distinct from Antigravity IDE storage.
- Add provider-aware CLI support for `list`, `inspect`, `filter`, `stats`, and `handoff`.
- Add `handoff --llm auto|always|never` with filtered-first behavior. The command always writes redacted `filtered.md` first and uses redacted filtered size to decide whether to call Local LLM.
- Add optional OpenAI-compatible Local LLM client configuration with deterministic defaults (`temperature: 0`) and configurable model/output/sampling settings.
- Add handoff JSON/Markdown artifacts with schema normalization, one repair attempt for invalid JSON, and derived-artifact disclosure.
- Record real local smoke tests for Claude Code, Codex, and Google Antigravity standalone app sessions.

## Deferred

Evidence store, stronger redaction, semantic evidence validation, MCP server, Skills, installer client selection, full docs, and full e2e validation are split into follow-up changes:

- `add-evidence-filtering-security`
- `add-session-context-mcp-server`
- `add-cross-agent-skills-installer`
- `add-session-context-docs-e2e`

## Impact

- Affected code: `cmd/cc-session/*`, `internal/session`, `internal/codexcodec`, `internal/antigravitycodec`, `internal/config`, `internal/redaction`, `internal/handoff`, `internal/distiller`.
- New runtime dependency: none for deterministic workflows. Local LLM is optional and used only when configured and selected by policy.
- No breaking changes to existing upstream Claude Code commands.
