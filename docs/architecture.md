# Architecture

This fork keeps the upstream `cc-session` CLI and analyzer pipeline intact while
adding provider-aware entry points for cross-agent session history.

## CLI Entry

`cmd/cc-session` is the executable surface. Claude Code remains the default
provider for existing commands, so current `list`, `read`, `context`, `stats`,
`audit`, `expand`, `inject`, and `benchmark` behavior stays compatible unless a
provider flag is passed.

Provider-aware commands currently implemented:

- `list --provider claude_code|codex|all`
- `inspect --provider auto|claude_code|codex`
- `filter --provider auto|claude_code|codex`
- `stats --provider claude_code|codex`

`antigravity` is reserved as a first-class provider name. The CLI also accepts
the user typo `angravity` as an alias, but the adapter intentionally fails with
an actionable "not implemented" message until a real Antigravity session format
is verified.

## Core Packages

- `internal/session`: shared legacy event model plus the new normalized provider
  model (`SessionProvider`, `SessionRef`, `SessionMetadata`, `SessionEvent`).
- `internal/claudecodec`: upstream Claude Code transcript parser.
- `internal/codexcodec`: Codex rollout JSONL parser. It reads real Codex session
  envelopes, normalizes them, and also maps them into the existing analyzer
  event model so deterministic filtering and stats can run without a Local LLM.
- `internal/analyzer`: deterministic raw/filtered text and category stats.
- `internal/summarizer`: existing tool-result compaction used by the analyzer.
- `internal/parser`: Claude Code session discovery and resolver.
- `internal/formatter`, `internal/inject`, `internal/benchmark`, `internal/tokens`:
  existing upstream behavior for rendering, injection, cost modeling, and token
  counting.

## Optional Local LLM Boundary

The deterministic path has no Local LLM dependency. A user with only Codex
sessions can run `list`, `inspect`, `filter`, and `stats -no-tokens` through the
Codex adapter. Local LLM support is a later distillation layer for generated
handoff artifacts, controlled by `local_llm.*` config, and must remain disabled
unless explicitly configured.
