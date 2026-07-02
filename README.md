# session-context-broker

> **A fork of [`Mapleeeeeeeeeee/cc-session-reader`](https://github.com/Mapleeeeeeeeeee/cc-session-reader)** (Apache-2.0).
> Upstream is a deterministic **Claude Code** session reader that compresses transcripts so past sessions can be reused cheaply. This fork keeps that core intact and extends it into a cross-agent **session context broker**.
>
> **What this fork adds on top of upstream:**
> - **More session sources** — Codex CLI and Google Antigravity standalone-app sessions alongside Claude Code, behind one normalized provider adapter (upstream is Claude Code only).
> - **Optional local-LLM handoff distillation** — distill a filtered transcript into a structured, evidence-referenced `handoff.json` via an OpenAI-compatible local endpoint. Optional by design: without a local LLM you still get filtered, evidence-backed artifacts.
> - **Evidence store** — filtered output, evidence index, and handoff artifacts persisted under `storage_root`, with on-demand evidence expansion.
> - **MCP server** — `cc-session serve-mcp` exposes the broker to Claude Code, Codex, and Antigravity as tools.
> - **Cross-agent skills** — installable resume / close / review-history workflows.
>
> Upstream CLI behavior (`list`, `read`, `context`, `inject`, `stats`, `expand`, `audit`) is preserved, and the project stays licensed under Apache-2.0.

Local LLM handoff distillation is optional. Users without a Local LLM can still list, inspect, filter, search, and create filtered evidence-backed handoff artifacts.

## Install

macOS / Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/Mapleeeeeeeeeee/cc-session-reader/main/install.sh | bash -s -- --clients claude,codex,antigravity
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/Mapleeeeeeeeeee/cc-session-reader/main/install.ps1 | iex
```

Non-interactive client selection:

```bash
./install.sh --clients all
./install.sh --clients none
./install.sh --clients claude,codex
```

```powershell
.\install.ps1 -Clients all
.\install.ps1 -Clients none
.\install.ps1 -Clients claude,codex
```

Install prompts show already-installed client integrations as checked. No separate `cc-session init` is required.

## Commands

| Command | Purpose |
|---|---|
| `list` | List sessions by provider |
| `inspect` | Show session metadata and stats |
| `filter` | Print deterministic filtered transcript |
| `handoff` | Write filtered/evidence artifacts and optional Local LLM handoff |
| `search` | Search evidence summaries |
| `evidence` | Expand one evidence ID with redaction by default |
| `verify-workspace` | Read-only git verification inside allowed roots |
| `serve-mcp` | Start stdio MCP server |
| `read`, `context`, `stats`, `audit`, `expand`, `inject`, `benchmark` | Preserved upstream commands |

Examples:

```bash
cc-session list --provider all -n 10
cc-session filter --provider codex <session-id>
cc-session handoff --provider antigravity --llm never <session-id>
cc-session handoff --provider codex --llm auto --config ~/.session-context/config.json <session-id>
cc-session serve-mcp --config ~/.session-context/config.json
```

## Configuration

Default path:

```text
~/.session-context/config.json
```

Override path:

```bash
export SESSION_CONTEXT_CONFIG=/path/to/config.json
```

Example:

```json
{
  "session_sources": {
    "claude_code": {"roots": ["~/.claude/projects"]},
    "codex": {"roots": ["~/.codex/sessions"]},
    "antigravity": {"roots": ["~/.gemini/antigravity/brain"]}
  },
  "storage_root": "~/.session-context",
  "allowed_workspace_roots": ["~/work", "D:/repo"],
  "local_llm": {
    "enabled": false,
    "base_url": "http://127.0.0.1:8000/v1",
    "api_key": "",
    "model": "Qwen3.6-35B-A3B",
    "max_context": 32000,
    "max_output_tokens": 4096,
    "timeout_seconds": 120,
    "min_filtered_chars": 8000,
    "temperature": 0,
    "top_p": 0.95,
    "top_k": 20
  }
}
```

Environment overrides include:

- `SESSION_CONTEXT_STORAGE_ROOT`
- `SESSION_CONTEXT_LOCAL_LLM_ENABLED`
- `LOCAL_LLM_BASE_URL`
- `LOCAL_LLM_API_KEY`
- `LOCAL_LLM_MODEL`
- `LOCAL_LLM_MAX_CONTEXT`
- `LOCAL_LLM_MAX_OUTPUT_TOKENS`
- `LOCAL_LLM_TIMEOUT_SECONDS`
- `LOCAL_LLM_MIN_FILTERED_CHARS`
- `LOCAL_LLM_TEMPERATURE`
- `LOCAL_LLM_TOP_P`
- `LOCAL_LLM_TOP_K`

## Output Artifacts

`cc-session handoff` writes under:

```text
storage_root/<provider>/<session-id>/
```

Artifacts:

- `manifest.json`
- `normalized.jsonl`
- `filtered.jsonl`
- `filtered.md`
- `evidence-index.json`
- `handoff.json` and `handoff.md` when Local LLM is used

Filtered artifacts and evidence expansion are redacted by default. Raw session files are never modified.

## Documentation

- [Architecture](docs/architecture.md)
- [Session Providers](docs/session-provider.md)
- [Normalized Event Schema](docs/normalized-event-schema.md)
- [Handoff Schema](docs/handoff-schema.md)
- [Local LLM Distillation](docs/local-llm-distillation.md)
- [MCP Tools](docs/mcp-tools.md)
- [Skills](docs/skills.md)
- [Security](docs/security.md)
- [Upstream Sync](docs/upstream-sync.md)
