# Upstream Sync

This fork preserves upstream deterministic filtering and existing Claude Code commands. Keep upstream changes isolated from fork-specific additions:

1. Rebase or merge upstream into a clean branch.
2. Run `go test ./...`.
3. Verify upstream commands still pass: `read`, `context`, `stats`, `audit`, `expand`, `inject`, `benchmark`.
4. Re-run provider smoke tests for Claude Code, Codex, and Google Antigravity standalone app.
5. Re-run `cc-session handoff --llm never` to verify deterministic filtered output remains independent of Local LLM.

Fork-specific areas are `internal/broker`, `internal/evidence`, provider adapters beyond Claude Code, MCP, Skills, and Local LLM distillation.
