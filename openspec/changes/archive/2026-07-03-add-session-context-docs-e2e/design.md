# Design: Docs, Config, and End-to-End Validation

## Decision

Treat this as a docs/tests/config polish pass with **no new runtime
behavior**, per its own scope boundary — everything it touches was already
implemented by the three preceding changes
(`add-session-context-mcp-server`, `add-cross-agent-skills-installer`,
`add-evidence-filtering-security`). Its job is to make the already-working
system legible and verifiable, not to change what it does.

## Documentation set

Nine focused docs under `docs/`, each covering one concern rather than one
combined reference, so a reader who only needs (say) the MCP tool contract
doesn't have to read the handoff schema to find it:

```
docs/architecture.md               overall pipeline
docs/session-provider.md           Claude Code / Codex / Antigravity formats
docs/normalized-event-schema.md    cross-provider event shape
docs/handoff-schema.md             handoff.json fields
docs/local-llm-distillation.md     optional LLM path, config
docs/mcp-tools.md                  the 9 MCP tools
docs/skills.md                     resume/close/review-history
docs/security.md                   redaction, path safety
docs/upstream-sync.md              staying in sync with the fork parent
docs/validation-report-2026-07-03.md  command-output evidence for task 3.3
```

`README.md` carries the fork-attribution note and top-level orientation;
the detailed "how" lives in `docs/`.

Note: `README.md` was substantially rewritten after this change's tasks
were checked off (retitled to `session-context-broker`, restructured to
Traditional-Chinese-primary with `README.en.md` as the English variant,
and given an explicit "how it works" pipeline section). The requirement
this change set — fork attribution, config coverage, provider identification
— still holds; the specific English-only draft it originally produced does
not exist anymore.

## Validation

Two test additions, both consistent with "Local LLM is optional, never a
default hard dependency" (from `add-evidence-filtering-security`):

- **Full pipeline e2e** (`cmd/cc-session/full_pipeline_test.go`): drives a
  fixture session through filter → evidence → handoff → verify_workspace,
  entirely offline.
- **Opt-in live Local LLM integration** (`internal/distiller/live_integration_test.go`):
  `t.Skip()`s unless `SESSION_CONTEXT_LIVE_LLM_TEST=1` is set, so
  `go test ./...` never requires a reachable Local LLM endpoint.

## Consequences

- Docs describe a system whose surfaces (MCP tool count/schemas, install
  registration behavior) have continued to change after this change was
  marked done; `docs/mcp-tools.md` and `docs/skills.md` should be revisited
  the next time those surfaces change materially, rather than treated as
  frozen.
