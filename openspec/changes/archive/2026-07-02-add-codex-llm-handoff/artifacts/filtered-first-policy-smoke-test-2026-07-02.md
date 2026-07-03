# Filtered-First Policy Smoke Test - 2026-07-02

## Scope

Validated the handoff policy after adding `--llm auto|always|never` and `local_llm.min_filtered_chars`.

Policy under test:

- Always write the deterministic redacted filtered artifact first.
- In `--llm auto`, compare redacted filtered chars against `local_llm.min_filtered_chars` (default `8000`).
- If below threshold, skip Local LLM and return filtered output.
- If above threshold and Local LLM is configured, call Local LLM.
- If Local LLM is absent in auto mode, keep filtered-only operation instead of failing.
- If `--llm always` is requested and Local LLM is absent, fail loud after writing filtered output.

## Commands

```powershell
go test ./...
go build ./...
openspec validate add-codex-llm-handoff --strict
go run ./cmd/cc-session handoff --provider claude_code --config (Join-Path $env:TEMP 'session-context-local-llm-live.json') --force 865e864d
go run ./cmd/cc-session handoff --provider codex --config (Join-Path $env:TEMP 'session-context-local-llm-live.json') --force 019f2314
go run ./cmd/cc-session handoff --provider codex --config (Join-Path $env:TEMP 'session-context-local-llm-live.json') --llm never --force 019f2314
go run ./cmd/cc-session handoff --provider antigravity --force whatever
```

## Results

### Unit/build/spec validation

- `go test ./...`: pass
- `go build ./...`: pass
- `openspec validate add-codex-llm-handoff --strict`: pass, `Change 'add-codex-llm-handoff' is valid`

### Claude Code short session, auto mode

Command:

```powershell
go run ./cmd/cc-session handoff --provider claude_code --config (Join-Path $env:TEMP 'session-context-local-llm-live.json') --force 865e864d
```

Result:

```text
Mode: filtered
Provider: claude_code
Session: 865e864d-0090-46b4-8363-45ebc5c5a19a
LLM policy: auto
LLM threshold: 8000
LLM decision: redacted filtered chars 3555 below threshold 8000
Raw chars: 32,110
Filtered chars: 2,220
Redacted input chars: 3,555
Filtered output: C:\Users\peter\.session-context\claude_code\865e864d-0090-46b4-8363-45ebc5c5a19a\filtered.md
```

Interpretation: short session correctly skipped Local LLM and wrote only filtered output.

### Codex long session, auto mode

Command:

```powershell
go run ./cmd/cc-session handoff --provider codex --config (Join-Path $env:TEMP 'session-context-local-llm-live.json') --force 019f2314
```

Result:

```text
Mode: llm
Provider: codex
Session: 019f2314-d889-7223-ae56-787c4d2cb8c0
LLM policy: auto
LLM threshold: 8000
LLM decision: redacted filtered chars 73722 meets threshold 8000
Model: local-model
Max context: 32000
Max output tokens: 12000
Temperature: 0
TopP: 0.95
TopK: 20
Chunks: 1
Repaired: true
Raw chars: 2,675,644
Filtered chars: 55,795
Redacted input chars: 73,722
Filtered output: C:\Users\peter\.session-context\codex\019f2314-d889-7223-ae56-787c4d2cb8c0\filtered.md
Output: C:\Users\peter\.session-context\codex\019f2314-d889-7223-ae56-787c4d2cb8c0
```

Artifact inspection:

```text
schema: session-context-handoff/v1
provider: codex
model: local-model
objective_len: 330
next_actions: 5
warnings: 4
conflicts: 0
handoff.md disclosure: present
filtered.md header/provider/session: present
```

Interpretation: long Codex session correctly used Local LLM in auto mode. The first model response required one repair round-trip; the repaired result validated and was written.

### Codex long session, forced filtered-only

Command:

```powershell
go run ./cmd/cc-session handoff --provider codex --config (Join-Path $env:TEMP 'session-context-local-llm-live.json') --llm never --force 019f2314
```

Result:

```text
Mode: filtered
Provider: codex
Session: 019f2314-d889-7223-ae56-787c4d2cb8c0
LLM policy: never
LLM threshold: 8000
LLM decision: --llm never requested
Raw chars: 2,677,582
Filtered chars: 55,971
Redacted input chars: 73,976
Filtered output: C:\Users\peter\.session-context\codex\019f2314-d889-7223-ae56-787c4d2cb8c0\filtered.md
```

Interpretation: users with Codex but no Local LLM can still use deterministic filtered output.

### Antigravity

Command:

```powershell
go run ./cmd/cc-session handoff --provider antigravity --force whatever
```

Result:

```text
Error: antigravity provider is recognized but session parsing is not implemented yet
exit status 1
```

Interpretation: provider alias is recognized, but Antigravity session parsing remains intentionally unimplemented in this slice.
