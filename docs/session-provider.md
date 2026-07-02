# Session Providers

Supported provider names:

- `claude_code`: Claude Code JSONL sessions under `~/.claude/projects`.
- `codex`: Codex rollout JSONL sessions under `~/.codex/sessions`.
- `antigravity`: Google Antigravity standalone app brain store under `~/.gemini/antigravity/brain`.

Aliases:

- `claude`, `claude-code`, `claude_code` -> `claude_code`
- `codex` -> `codex`
- `antigravity`, `angravity` -> `antigravity`

Antigravity here means the Google standalone app brain store, not an IDE-like Antigravity storage layout.

Provider adapters expose `Discover`, `Inspect`, and `Parse` over normalized `session.SessionEvent` records. CLI commands also map events into the legacy analyzer model so upstream filtering behavior remains compatible.
