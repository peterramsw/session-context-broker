# Antigravity Standalone Provider Smoke Test - 2026-07-02

## Scope

Implemented and validated Google Antigravity standalone app session support, targeting the standalone app's local brain store, not the Antigravity IDE storage.

Local root used:

```text
C:\Users\peter\.gemini\antigravity\brain
```

Session transcript shape:

```text
C:\Users\peter\.gemini\antigravity\brain\<conversation-id>\.system_generated\logs\transcript.jsonl
C:\Users\peter\.gemini\antigravity\brain\<conversation-id>\.system_generated\logs\transcript_full.jsonl
```

The codec prefers `transcript_full.jsonl` when present, falling back to `transcript.jsonl`.

## Commands

```powershell
go test ./internal/antigravitycodec ./cmd/cc-session
go run ./cmd/cc-session list --provider antigravity -n 3
go run ./cmd/cc-session inspect --provider antigravity 507ca213
go run ./cmd/cc-session stats --provider antigravity --no-tokens 507ca213
go run ./cmd/cc-session handoff --provider antigravity --llm never --force 507ca213
go run ./cmd/cc-session filter --provider antigravity 507ca213 | Select-Object -First 20
go test ./...
go build ./...
```

## Results

### Unit and CLI tests

```text
ok   github.com/Mapleeeeeeeeeee/cc-session-reader/internal/antigravitycodec
ok   github.com/Mapleeeeeeeeeee/cc-session-reader/cmd/cc-session
```

### Real session list

```text
507ca213-cb36-4d71-8fe3-e15d5431ac44  06-30 09:29  android                    [antigravity]  我想詢問，功能列上的圖標(剪貼簿)，可以做 上滑=貼上與下滑=複製 嗎？
5863cf99-e94f-4504-9176-c6876258d7a5  06-29 17:06  ?                          [antigravity]  Research the Zhuyin phrase/trigram bin file structure and the build pipeline....
3027bb91-76bd-4bf3-b7bd-a769a7dac61d  06-29 17:05  typingnote                 [antigravity]  唯讀分析 目前codex在做add-boshiamy-continuous-ime，你只能做唯讀不可以修改任何東西 codex一直改不好。而我有個想法… ...
```

### Real session inspect

```text
Provider: antigravity
Session: 507ca213-cb36-4d71-8fe3-e15d5431ac44
Path: C:\Users\peter\.gemini\antigravity\brain\507ca213-cb36-4d71-8fe3-e15d5431ac44\.system_generated\logs\transcript_full.jsonl
CWD: d:\repo\typingnote\android
Started: 2026-06-30T09:29:15Z
Messages: user=2 assistant=2
Tools: calls=25 results=25
Raw chars: 78,400
Filtered chars: 3,077
Saved: 75,323 (96.1%)
```

### Real session stats

```text
Session: 507ca213
Transcript: 92.2KB

=== Characters ===
  Raw:          78,400
  Filtered:      3,077
  Saved:        75,323 (96.1%)
```

Tool breakdown excerpt:

```text
  view_file          9 calls       1,177 input      37,666 result
  run_command       10 calls       2,629 input      24,586 result
  write_to_file      1 calls       5,517 input           0 result
  list_dir           4 calls         610 input       1,186 result
  call_mcp_tool      1 calls         145 input         206 result
  code_action        0 calls           0 input         326 result
```

### Real session handoff, filtered-only

```text
Mode: filtered
Provider: antigravity
Session: 507ca213-cb36-4d71-8fe3-e15d5431ac44
LLM policy: never
LLM threshold: 8000
LLM decision: --llm never requested
Raw chars: 78,400
Filtered chars: 3,077
Redacted input chars: 4,474
Filtered output: C:\Users\peter\.session-context\antigravity\507ca213-cb36-4d71-8fe3-e15d5431ac44\filtered.md
```

### Real session filter excerpt

```text
我想詢問，功能列上的圖標(剪貼簿)，可以做 上滑=貼上與下滑=複製 嗎？
[run_command]
 -> ok: Created At: 2026-06-30T09:29:18Z
[call_mcp_tool]
 -> FAILED: Created At: 2026-06-30T09:29:19Z
[view_file]
 -> ok: Created At: 2026-06-30T09:29:20Z
```

### Full validation

```text
go test ./...: pass
go build ./...: pass
```

## Notes

- The implementation does not parse Antigravity IDE's VS Code-style storage.
- The standalone app adapter is intentionally lightweight: JSONL transcript parsing only, no SQLite dependency.
- `ERROR_MESSAGE` and explicit execution-failure markers are treated as failed tool results. File contents containing words like "error" or "failed" are not treated as execution failure by themselves.
