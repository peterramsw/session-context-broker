# Architecture

`cc-session` keeps deterministic filtering as the default path. Local LLM use is optional and only runs after the filtered transcript has already been produced and redacted.

## Layers

- `cmd/cc-session`: CLI commands and stdio MCP entry point.
- `internal/broker`: shared session operations used by CLI and MCP.
- `internal/session`: provider-neutral session refs, metadata, normalized events, and legacy event model.
- `internal/claudecodec`, `internal/codexcodec`, `internal/antigravitycodec`: provider adapters.
- `internal/analyzer`: deterministic raw/filtered text and stats.
- `internal/evidence`: manifest, normalized events, filtered artifact, evidence index, and bounded expansion.
- `internal/distiller`: optional OpenAI-compatible Local LLM handoff generation.
- `internal/handoff`: structured handoff schema, markdown rendering, and validation.

## Data Flow

1. Provider adapter discovers and parses a session.
2. Analyzer produces deterministic filtered text.
3. Redaction runs before filtered artifacts, Local LLM requests, and evidence expansion responses.
4. Evidence store writes derived artifacts under `storage_root/<provider>/<session-id>/`.
5. Optional Local LLM generates `handoff.json` and `handoff.md`; validator strips unknown evidence refs and demotes unsafe claims.

Raw session files are never mutated.
