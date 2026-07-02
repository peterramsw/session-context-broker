# Original cc-session stats baseline - 2026-07-02

Purpose: establish a pre-Qwen comparison baseline using the original deterministic `cc-session stats -no-tokens` workflow. These measurements are character-count based and do not use an LLM or token-counting API.

Prior Claude-provided examples (`b25b1404`, `58309d91`) were intentionally excluded from this baseline. The two sessions below were selected from recent `cc-session list -n 20` output as larger, tool-I/O-heavy Claude Code sessions.

## Commands

```powershell
go run ./cmd/cc-session stats 6881db75 -no-tokens
go run ./cmd/cc-session stats 20ad80d4 -no-tokens
```

## Baseline Results

| Session | Project | Transcript | Raw chars | Filtered chars | Saved chars | Saved |
|---|---|---:|---:|---:|---:|---:|
| `6881db75` | `D--repo-typingnote` | 878.7 KB | 236,136 | 22,554 | 213,582 | 90.4% |
| `20ad80d4` | `D--repo-typingnote` | 1751.2 KB | 391,997 | 80,579 | 311,418 | 79.4% |

## Signal Breakdown

| Session | User text kept | Assistant text kept | Tool summaries kept | Tool input cut | Tool result cut | Command noise cut |
|---|---:|---:|---:|---:|---:|---:|
| `6881db75` | 8,644 | 10,291 | 6,890 | 16,980 | 199,396 | 145 |
| `20ad80d4` | 30,903 | 40,665 | 8,815 | 149,849 | 170,320 | 0 |

## Model Context Metadata

| Session | Last turn context | Total output | API calls |
|---|---:|---:|---:|
| `6881db75` | 196,951 | 52,452 | 53 |
| `20ad80d4` | 340,361 | 179,134 | 68 |

## Notes For Qwen Comparison

- Use the same two session IDs when comparing the future Qwen handoff output.
- The relevant comparison points are:
  - deterministic filtered size from this baseline,
  - Qwen handoff JSON/Markdown size,
  - count of claims requiring re-verification,
  - validation warnings/conflicts,
  - evidence expansion needed during resume.
- The expected Qwen win is not raw compression ratio alone. It should reduce resume effort by turning the filtered transcript into structured, evidence-referenced state.
