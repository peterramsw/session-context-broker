# Design: Cross-Agent Skills Installer

## Decision

Author each workflow once under `skills/common/*.md` and keep every
per-platform `SKILL.md` a thin wrapper that points at the shared content,
rather than writing three full copies of Resume/Close/Review-History. MCP
gives an agent tools; Skills tell it when and how to use them safely — the
two are complementary, not overlapping (an MCP tool call has no built-in
policy against trusting an unverified "tests passed" claim; the Skill
carries that policy).

## Architecture

```
skills/
  common/
    resume-session.md    ← full workflow content, written once
    close-session.md
    review-history.md
  claude-code/cc-session/SKILL.md   ← frontmatter + pointer to common/
  codex/cc-session/SKILL.md         ← same pattern, Codex's skill convention
  antigravity/cc-session/SKILL.md   ← same pattern, Antigravity's convention
```

Each wrapper carries only platform-specific frontmatter (name, description,
allowed-tools) and a short instruction to use the shared `common/*.md`
files installed beside it. Editing a shared workflow updates every
platform without touching the wrappers.

## Installer

`install.sh` / `install.ps1` gained a client selector:

- `--clients all|none|claude,codex,antigravity` for non-interactive installs.
- Interactive mode shows each client's current install state
  (`[x]`/`[ ]`) before prompting, so re-running the installer is safe and
  legible.
- No separate `cc-session init` step — selecting a client during install is
  sufficient to make it usable.
- (Landed after this change, in the same install flow) client selection also
  drives MCP registration for Codex (`config.toml`) and Antigravity
  (`mcp_config.json`), so a client that gets its skill also gets the MCP
  server wired up — see `add-session-context-mcp-server`.

## Verification

- Install paths confirmed for real: `~/.claude/skills/cc-session`,
  `~/.codex/skills/cc-session`, `~/.gemini/antigravity/skills/cc-session`.
- Live-tested: an Antigravity session without the skill installed had to
  reverse-engineer `cc-session` from scratch (web search, `--help`,
  trial and error) with no safety policy applied. This is the concrete
  evidence for why the Skill layer matters beyond raw MCP tool access.

## Consequences

- This change intentionally adds no new parser or evidence-store behavior —
  it only packages the existing CLI/MCP surface for agent use.
- Keeping wrappers thin means a broken shared workflow breaks all three
  platforms identically (no per-platform drift to debug), which is the
  intended trade-off.
