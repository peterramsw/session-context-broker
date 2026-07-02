# Provider smoke test - 2026-07-02

Purpose: record the first provider-level smoke test results for the
`add-codex-llm-handoff` implementation slice. These checks do not use a Local
LLM endpoint.

Environment:

- Workspace: `D:\repo\session-context-broker`
- Branch: `main`
- Date: 2026-07-02

## Codex

Command:

```powershell
go run ./cmd/cc-session list --provider codex -n 1
```

Result: pass. The CLI discovered the real local Codex session
`019f2314-d889-7223-ae56-787c4d2cb8c0` under `~/.codex/sessions`.

Command:

```powershell
go run ./cmd/cc-session inspect --provider codex 019f2314
```

Result: pass.

Key output:

```text
Provider: codex
Session: 019f2314-d889-7223-ae56-787c4d2cb8c0
Path: C:\Users\peter\.codex\sessions\2026\07\02\rollout-2026-07-02T21-46-41-019f2314-d889-7223-ae56-787c4d2cb8c0.jsonl
CWD: D:\repo\session-context-broker
Messages: user=9 assistant=91
Tools: calls=203 results=205
Raw chars: 1,580,363
Filtered chars: 34,258
Saved: 1,546,105 (97.8%)
```

Command:

```powershell
go run ./cmd/cc-session stats --provider codex 019f2314 -no-tokens
```

Result: pass.

Key output:

```text
Raw:       1,580,146
Filtered:     34,237
Saved:     1,545,909 (97.8%)
```

Note: this session was still active while testing, so raw/filtered counts can
increase between runs.

## Claude Code

Command:

```powershell
go run ./cmd/cc-session list -n 1
```

Result: pass. The default provider path remains Claude Code and discovered the
real local Claude session `865e864d-0090-46b4-8363-45ebc5c5a19a`.

Command:

```powershell
go run ./cmd/cc-session inspect --provider claude_code 865e864d
```

Result: pass.

Key output:

```text
Provider: claude_code
Session: 865e864d-0090-46b4-8363-45ebc5c5a19a
Path: C:\Users\peter\.claude\projects\D--repo-session-context-broker\865e864d-0090-46b4-8363-45ebc5c5a19a.jsonl
Transcript: 161.7KB
Raw chars: 32,110
Filtered chars: 2,220
```

## Antigravity

Command:

```powershell
go run ./cmd/cc-session list --provider antigravity -n 1
```

Result: expected failure. The provider is recognized, but the session parser is
not implemented yet because the real local Antigravity session format still
needs verification.

Output:

```text
Error: antigravity provider is recognized but session parsing is not implemented yet
exit status 1
```

Command:

```powershell
go run ./cmd/cc-session list --provider angravity -n 1
```

Result: expected failure. The user typo `angravity` is accepted as an alias for
`antigravity` and reaches the same explicit implementation-status error.

Output:

```text
Error: antigravity provider is recognized but session parsing is not implemented yet
exit status 1
```

## Regression checks

Commands:

```powershell
go test ./...
go build ./...
openspec validate add-codex-llm-handoff --strict
```

Result: pass.

Key output:

```text
ok  	github.com/Mapleeeeeeeeeee/cc-session-reader/cmd/cc-session	(cached)
ok  	github.com/Mapleeeeeeeeeee/cc-session-reader/internal/codexcodec	(cached)
ok  	github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session	(cached)
go build ./... exited 0
Change 'add-codex-llm-handoff' is valid
```

## Status

- Claude Code: real-session smoke passed.
- Codex: real-session smoke passed.
- Antigravity: provider-name and alias smoke passed as expected failure;
  real-session parsing is not implemented yet.
