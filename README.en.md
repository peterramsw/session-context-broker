# session-context-broker

[繁體中文](README.md) ｜ **English**

> **A fork of [`Mapleeeeeeeeeee/cc-session-reader`](https://github.com/Mapleeeeeeeeeee/cc-session-reader), released under the upstream's Apache License 2.0.**
> Upstream is a deterministic (no-LLM) **Claude Code** session reader that compresses the heavy tool noise out of a transcript while keeping the conversation, so past sessions can be reloaded with very few tokens. This fork keeps that core and extends it into a cross-agent **session context broker**.

## What this fork adds on top of upstream

- **More session sources** — besides Claude Code, it reads **Codex CLI** and **Google Antigravity 2.0** sessions, all behind one normalized provider adapter (upstream is Claude Code only).
- **Optional local-LLM handoff distillation** — distill a filtered transcript into a structured, evidence-referenced `handoff.json` via an OpenAI-compatible local endpoint. **Off by default**: without a local LLM you still get filtering and evidence artifacts.
- **Evidence store** — filtered output, evidence index, and handoff artifacts persist under `storage_root`, expandable on demand by evidence ID.
- **MCP server** — `cc-session serve-mcp` exposes the broker to Claude Code / Codex / Antigravity as tools.
- **Cross-agent skills** — installable resume / close / review-history workflows.

All upstream CLI behavior (`list`, `read`, `context`, `inject`, `stats`, `expand`, `audit`) is preserved.

## How it works (pipeline)

The core flow is "filter deterministically first, then optionally hand off to a local LLM." The raw session is never modified:

```
raw session (Claude Code / Codex / Antigravity)
  → deterministic filter   ← drop tool noise, keep conversation + risk signals (error/rollback/exit code…)
  → secret redaction       ← mask API keys, tokens, passwords by default
  → evidence index         ← each compressed chunk gets a resolvable evidence_id
  → [optional] local LLM   ← produce a structured handoff.json (objective / decisions / next_actions…)
  → schema + evidence check ← unevidenced claims are demoted, never treated as "confirmed"
  → MCP / Skill            ← served to a fresh session to resume from
```

Two layers of value, kept distinct:

- **Deterministic filtering (no LLM)** — tool-heavy sessions typically compress **80–88%** (pure discussion or large plan docs compress less). This layer has **no hallucination risk** and is where the token savings come from.
- **Local-LLM distillation (optional)** — not about saving more tokens, but about turning a long session into a navigable structure (objective, next steps, re-verify checklist). Best for resuming tool-heavy engineering sessions. The handoff is a **derived artifact, not a source of truth**; unevidenced claims are demoted to `claims_requiring_reverification`.

> In short: token savings come from the filtering layer; the local LLM is a "navigation" enhancement you turn on when needed.

### When does it actually run?

**Nothing runs in the background and there is no timer.** Once installed, this tool does nothing on its own and has zero effect on your current conversation's token usage. It only runs when:

- **Intent-triggered (default)** — the Skill's `description` lets Claude Code / Codex / Antigravity recognize when to reach for it. When you naturally say "continue from where we left off," "what did we do last time," or "wrap up this session," the agent decides to call `cc-session` on its own — you don't need to memorize any command.
- **Manual** — you can also say it directly: "run `cc-session list`" or ask the agent to call a specific MCP tool.

It operates across sessions (loading an old conversation cheaply into a new one), not on the conversation you're currently having — your current token usage is unaffected.

## Install

### One-liner

The installer downloads the platform binary and can install the Claude Code / Codex / Antigravity skills at the same time.

**macOS / Linux**

```bash
curl -fsSL https://raw.githubusercontent.com/peterramsw/session-context-broker/main/install.sh | bash -s -- --clients claude,codex,antigravity
```

**Windows PowerShell**

```powershell
irm https://raw.githubusercontent.com/peterramsw/session-context-broker/main/install.ps1 | iex
```

### Choosing which clients to install

`--clients` accepts `all`, `none`, or a comma list of `claude,codex,antigravity`. Interactive mode shows already-installed clients as checked.

```bash
./install.sh --clients all                    # all three
./install.sh --clients claude                 # Claude Code only
./install.sh --clients codex,antigravity      # Codex + Antigravity only
./install.sh --no-skill                        # binary only, no skills
```

```powershell
.\install.ps1 -Clients all
.\install.ps1 -Clients claude
.\install.ps1 -Clients codex,antigravity
```

Skill install locations per client:

| Client | Skill path |
|---|---|
| Claude Code | `~/.claude/skills/cc-session` |
| Codex | `~/.codex/skills/cc-session` |
| Google Antigravity 2.0 | `~/.gemini/antigravity/skills/cc-session` |

### Other install methods

- **Releases** — download the platform archive from [GitHub Releases](https://github.com/peterramsw/session-context-broker/releases) and put `cc-session` on your PATH.
- **From source** — `git clone`, then `go build ./cmd/cc-session`. (Note: `go install` currently pulls the **upstream** version because the module path still follows upstream `github.com/Mapleeeeeeeeeee/cc-session-reader`.)

## Usage

### CLI subcommands

| Command | Purpose |
|---|---|
| `list` | List sessions by provider |
| `inspect` | Show session metadata and stats |
| `filter` | Print the deterministic filtered transcript |
| `handoff` | Write filtered/evidence artifacts and optionally a local-LLM handoff |
| `search` | Search evidence summaries |
| `evidence` | Expand one evidence ID (redacted by default) |
| `verify-workspace` | Read-only git checks inside allowed roots |
| `serve-mcp` | Start the stdio MCP server |
| `read`, `context`, `stats`, `audit`, `expand`, `inject`, `benchmark` | Preserved upstream commands |

Examples:

```bash
cc-session list --provider all -n 10
cc-session filter --provider codex <session-id>

# No local LLM: filtered + evidence artifacts only
cc-session handoff --provider antigravity --llm never <session-id>

# With local LLM: distill only past a threshold (auto), or force it (always)
cc-session handoff --provider codex --llm auto <session-id>

cc-session serve-mcp --config ~/.session-context/config.json
```

### Wiring up MCP

Any MCP-capable client launches the same stdio server: `cc-session serve-mcp`. Example Claude Code project `.mcp.json`:

```json
{
  "mcpServers": {
    "cc-session": {
      "command": "cc-session",
      "args": ["serve-mcp"]
    }
  }
}
```

Codex and Antigravity point at the same `cc-session serve-mcp` command in their own MCP config formats. The tools exposed (`list_sessions`, `inspect_session`, `filter_session`, `create_handoff`, `get_handoff`, `search_session`, `expand_evidence`, `compare_context_size`, `verify_workspace`) are documented in [docs/mcp-tools.md](docs/mcp-tools.md).

## Configuration

Default path `~/.session-context/config.json`, overridable with `SESSION_CONTEXT_CONFIG`. **The file is optional** — without it you can still list/inspect/filter/search; it just won't enable the local LLM.

```json
{
  "session_sources": {
    "claude_code": {"roots": ["~/.claude/projects"]},
    "codex": {"roots": ["~/.codex/sessions"]},
    "antigravity": {"roots": ["~/.gemini/antigravity/brain"]}
  },
  "storage_root": "~/.session-context",
  "allowed_workspace_roots": ["~/work"],
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

Environment variables can override config, including `SESSION_CONTEXT_STORAGE_ROOT`, `SESSION_CONTEXT_LOCAL_LLM_ENABLED`, `LOCAL_LLM_BASE_URL`, `LOCAL_LLM_API_KEY`, `LOCAL_LLM_MODEL`, `LOCAL_LLM_MAX_CONTEXT`, `LOCAL_LLM_MAX_OUTPUT_TOKENS`, `LOCAL_LLM_TIMEOUT_SECONDS`, `LOCAL_LLM_MIN_FILTERED_CHARS`, `LOCAL_LLM_TEMPERATURE`, `LOCAL_LLM_TOP_P`, `LOCAL_LLM_TOP_K`.

## Output artifacts

`cc-session handoff` writes under `storage_root/<provider>/<session-id>/`:

- `manifest.json`, `normalized.jsonl`, `filtered.jsonl`, `filtered.md`, `evidence-index.json`
- `handoff.json` and `handoff.md` (only when the local LLM is used)

Filtered artifacts and evidence expansion are redacted by default; **raw session files are never modified**.

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

## License

Apache License 2.0, **inherited from upstream** `Mapleeeeeeeeeee/cc-session-reader`. The `LICENSE` file is unchanged; this fork's additions (Codex/Antigravity support, local-LLM handoff, MCP, and skills) are released under the same Apache-2.0. See [LICENSE](LICENSE).
