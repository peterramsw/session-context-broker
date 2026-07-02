# Local LLM smoke test - 2026-07-02

Purpose: record the first Local LLM handoff implementation and live smoke test
for `add-codex-llm-handoff`.

No API key or secret value is recorded in this artifact.

## Implementation Scope

Implemented in this slice:

- `internal/config`: loads `SESSION_CONTEXT_CONFIG` or
  `~/.session-context/config.json`; supports new `local_llm` config and legacy
  `qwen` config compatibility.
- `internal/redaction`: masks common API key, token, password, cookie, JWT, and
  private-key shapes before Local LLM input.
- `internal/handoff`: handoff Go structs, JSON normalization/validation,
  Markdown rendering, disclosure sentence, and atomic `handoff.json` /
  `handoff.md` writes.
- `internal/distiller`: OpenAI-compatible `/chat/completions` client, filtered
  transcript prompt, line-preserving chunking, one JSON repair attempt, and
  mock-server tests.
- `cmd/cc-session handoff`: provider-aware CLI command that generates and writes
  Local LLM handoff artifacts.

Not completed in this slice:

- Full evidence store / evidence-index expansion.
- MCP server.
- Antigravity session parsing.
- Phase-aware chunk labels beyond line-preserving chunks.
- Full redaction high-entropy heuristic.

## Model Parameter Policy

For this handoff/distillation use case, deterministic generation is the default:

- `temperature` defaults to `0` and is always sent in the chat-completions
  request.
- `max_output_tokens` is sent as OpenAI-compatible `max_tokens`.
- `max_context` is a client-side chunking budget. It does not and cannot change
  vLLM server-side `--max-model-len`; that remains a server deployment
  property.
- `top_p` and `top_k` are config/env controlled and are only sent when set, so a
  non-vLLM OpenAI-compatible endpoint is not forced to accept vLLM-specific
  sampling fields.

Temporary live-test config was generated outside the repo at:

```text
C:\Users\peter\AppData\Local\Temp\session-context-local-llm-live.json
```

It reused the existing local endpoint/model from the user's private
`~/.session-context/config.json`, but added:

```json
{
  "temperature": 0,
  "top_p": 0.95,
  "top_k": 20
}
```

## Token Budget Check

`internal/tokens` was inspected. It is an Anthropic count-tokens API client, not
a local tokenizer. This slice therefore does not add a tokenizer dependency.
`local_llm.max_context` is currently enforced as a conservative client-side
character budget for chunking.

## Regression Tests

Command:

```powershell
go test ./...
```

Result: pass.

Key packages added or updated:

```text
ok  	github.com/Mapleeeeeeeeeee/cc-session-reader/cmd/cc-session
ok  	github.com/Mapleeeeeeeeeee/cc-session-reader/internal/config
ok  	github.com/Mapleeeeeeeeeee/cc-session-reader/internal/distiller
ok  	github.com/Mapleeeeeeeeeee/cc-session-reader/internal/handoff
ok  	github.com/Mapleeeeeeeeeee/cc-session-reader/internal/redaction
```

Command:

```powershell
go build ./...
openspec validate add-codex-llm-handoff --strict
```

Result: pass.

## Claude Code Live Handoff

Command:

```powershell
go run ./cmd/cc-session handoff --provider claude_code --config $env:TEMP\session-context-local-llm-live.json --force 865e864d
```

Result: pass.

Output:

```text
Provider: claude_code
Session: 865e864d-0090-46b4-8363-45ebc5c5a19a
Model: q36-35b-general/ornith
Max context: 32000
Max output tokens: 12000
Temperature: 0
TopP: 0.95
TopK: 20
Chunks: 1
Repaired: false
Raw chars: 32,110
Filtered chars: 2,220
Redacted input chars: 3,555
Output: C:\Users\peter\.session-context\claude_code\865e864d-0090-46b4-8363-45ebc5c5a19a
```

Earlier legacy-config smoke for this same Claude session also passed and proved
the one-repair flow:

```text
Repaired: true
```

## Codex Live Handoff

Command:

```powershell
go run ./cmd/cc-session handoff --provider codex --config $env:TEMP\session-context-local-llm-live.json --force 019f2314
```

Result: pass after schema-tolerance fixes.

Output:

```text
Provider: codex
Session: 019f2314-d889-7223-ae56-787c4d2cb8c0
Model: q36-35b-general/ornith
Max context: 32000
Max output tokens: 12000
Temperature: 0
TopP: 0.95
TopK: 20
Chunks: 1
Repaired: false
Raw chars: 2,145,922
Filtered chars: 44,269
Redacted input chars: 58,684
Output: C:\Users\peter\.session-context\codex\019f2314-d889-7223-ae56-787c4d2cb8c0
```

The current Codex session was active while testing, so character counts can
increase between runs.

During live testing, the Local LLM produced schema-near output with string
claims/actions and object-shaped `workflow_improvement_candidates`. The parser
now normalizes those into the canonical schema instead of failing the whole
handoff.

## Codex Artifact Checks

Command:

```powershell
$p='C:\Users\peter\.session-context\codex\019f2314-d889-7223-ae56-787c4d2cb8c0\handoff.json'
$j=Get-Content $p -Raw | ConvertFrom-Json
```

Result:

```text
schema=session-context-handoff/v1 provider=codex model=q36-35b-general/ornith objective_len=245 next_actions=4 workflow_candidates=3 warnings=3 conflicts=0
```

Disclosure check:

```text
codex_disclosure=present
```

Secret scan over generated `handoff.json` + `handoff.md`:

```text
codex_secret_scan=passed
```

## Antigravity Handoff Path

Command:

```powershell
go run ./cmd/cc-session handoff --provider antigravity --force whatever
```

Result: expected failure. Antigravity is recognized, but real session parsing is
not implemented yet, so the CLI does not call the Local LLM.

Output:

```text
Error: antigravity provider is recognized but session parsing is not implemented yet
exit status 1
```
