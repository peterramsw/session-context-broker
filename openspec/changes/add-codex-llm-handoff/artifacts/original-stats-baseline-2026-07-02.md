# Codex original-style stats baseline - 2026-07-02

Purpose: establish a pre-Local LLM comparison baseline for real Codex session files using the same policy as the original `cc-session stats -no-tokens` workflow: keep user/assistant natural-language text, replace tool calls/results with deterministic one-line summaries, and count dropped tool I/O / system noise separately.

The upstream `cc-session stats` command cannot read Codex sessions yet, so these measurements were produced with a one-off parser over `C:\Users\peter\.codex\sessions\**\rollout-*.jsonl`. Raw session content was not copied into this artifact.

## Codex Session Files

| Session | JSONL file | Snapshot size | JSONL lines |
|---|---|---:|---:|
| `019f2314-d889-7223-ae56-787c4d2cb8c0` | `C:\Users\peter\.codex\sessions\2026\07\02\rollout-2026-07-02T21-46-41-019f2314-d889-7223-ae56-787c4d2cb8c0.jsonl` | 906.5 KB | 368 |
| `019f220d-c6ff-7fe0-904e-a19cecbb2edf` | `C:\Users\peter\.codex\sessions\2026\07\02\rollout-2026-07-02T16-59-21-019f220d-c6ff-7fe0-904e-a19cecbb2edf.jsonl` | 720.8 KB | 166 |

## Baseline Results

| Session | Raw chars | Filtered chars | Saved chars | Saved |
|---|---:|---:|---:|---:|
| `019f2314-d889-7223-ae56-787c4d2cb8c0` | 803,535 | 25,849 | 777,686 | 96.8% |
| `019f220d-c6ff-7fe0-904e-a19cecbb2edf` | 634,113 | 41,834 | 592,279 | 93.4% |

## Breakdown

| Session | User text kept | Assistant text kept | Tool summaries kept | Tool input cut | Tool result cut | System noise cut | Reasoning noise cut | Event-stream noise cut |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `019f2314-d889-7223-ae56-787c4d2cb8c0` | 16,576 | 17,996 | 8,342 | 26,646 | 319,808 | 257,757 | 129,071 | 52,600 |
| `019f220d-c6ff-7fe0-904e-a19cecbb2edf` | 64,980 | 7,988 | 5,243 | 8,973 | 430,517 | 91,242 | 35,286 | 31,446 |

## Per-tool Summary

| Session | Tool | Calls | Input chars | Result chars |
|---|---|---:|---:|---:|
| `019f2314-d889-7223-ae56-787c4d2cb8c0` | `shell_command` | 82 | 23,584 | 312,327 |
| `019f2314-d889-7223-ae56-787c4d2cb8c0` | `tool_search_call` | 1 | 76 | 6,108 |
| `019f2314-d889-7223-ae56-787c4d2cb8c0` | `apply_patch` | 3 | 2,839 | 354 |
| `019f2314-d889-7223-ae56-787c4d2cb8c0` | `codegraph_status` | 3 | 147 | 1,019 |
| `019f220d-c6ff-7fe0-904e-a19cecbb2edf` | `shell_command` | 36 | 7,748 | 411,322 |
| `019f220d-c6ff-7fe0-904e-a19cecbb2edf` | `tool_search_call` | 1 | 89 | 8,587 |
| `019f220d-c6ff-7fe0-904e-a19cecbb2edf` | `codegraph_node` | 2 | 215 | 7,345 |
| `019f220d-c6ff-7fe0-904e-a19cecbb2edf` | `codegraph_search` | 6 | 588 | 1,917 |
| `019f220d-c6ff-7fe0-904e-a19cecbb2edf` | `codegraph_context` | 1 | 286 | 927 |
| `019f220d-c6ff-7fe0-904e-a19cecbb2edf` | `codegraph_status` | 1 | 47 | 419 |

## Method Notes For Future Codex Adapter

- `response_item.message` with role `user` or `assistant` is treated as canonical conversation text and retained.
- `response_item.function_call`, `custom_tool_call`, and `tool_search_call` arguments are counted as raw tool input and replaced with summaries.
- `response_item.function_call_output`, `custom_tool_call_output`, and `tool_search_output` are counted as raw tool result and replaced with summaries.
- `session_meta`, `turn_context`, `developer` messages, `event_msg` status stream, and encrypted `reasoning` payloads are counted as raw noise dropped from the deterministic filtered view.
- The future implemented Codex adapter should reproduce this policy with tests, then replace this one-off measurement with CLI/MCP-produced stats.

## Notes For Local LLM Comparison

- Use the same two Codex session IDs when comparing future Local LLM handoff output.
- Compare deterministic filtered size from this baseline against Local LLM handoff JSON/Markdown size, validation warnings/conflicts, claims requiring re-verification, and evidence expansion needed during resume.
- The expected Local LLM win is not raw compression ratio alone. It should reduce resume effort by turning the filtered transcript into structured, evidence-referenced state.
