# Session Providers

The provider layer lets the fork read sessions from multiple coding agents
without hardwiring every command to one transcript format.

## Provider Names

- `claude_code`: existing upstream Claude Code sessions.
- `codex`: Codex rollout JSONL sessions.
- `antigravity`: reserved first-class provider target; adapter pending real
  local format verification.

Provider aliases accepted by the CLI:

- `claude`, `claude-code`, `claude_code` -> `claude_code`
- `codex` -> `codex`
- `antigravity`, `angravity` -> `antigravity`

## Codex Format Target

The Codex adapter targets the local rollout JSONL shape observed under
`~/.codex/sessions`. Each line is a JSON envelope with:

- `timestamp`
- `type`
- `payload`

Handled envelope types:

- `session_meta`: session ID, cwd, originator, CLI version, model provider.
- `event_msg`: user and agent messages, plus MCP tool result events.
- `response_item`: message, function/tool call, function/tool output, reasoning,
  and unknown future shapes.

Unknown Codex event shapes are preserved as `event_type: "unknown"` with the
compact original payload in `content`. Malformed JSONL lines are returned as
parse errors without aborting the rest of the file.

## Discovery

Codex discovery checks roots in this order:

1. Explicit `codexcodec.Codec{Roots: ...}` values.
2. `CODEX_SESSION_ROOTS` environment variable.
3. Platform default `~/.codex/sessions`.

`CODEX_SESSION_ROOTS` accepts comma, semicolon, or OS path-list separators.

Claude Code discovery still uses the existing upstream parser store. A shared
config-file discovery layer across all providers is still part of the active
OpenSpec change and is not completed in this first implementation slice.

## Normalized Event IDs

`internal/session.StableEventID` derives event IDs from provider, session ID,
sequence, source path, source line, source byte offset, and event type. It does
not include filtered content or content hashes, so evidence references stay
stable when summarization or redaction changes.
