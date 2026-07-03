# Skills

## Trigger mechanism

There is no background job or timer. A Skill only runs when the host agent
(Claude Code, Codex, Antigravity) matches the user's natural-language intent
against the Skill's frontmatter `description` — e.g. "resume where we left
off," "close out this session," "review past corrections" — and decides to
load it. The user can also trigger it explicitly (e.g. asking the agent to
run a specific `cc-session` command or MCP tool). Neither path affects the
token usage of the conversation currently in progress; the Skill/MCP surface
only loads *other* (past) sessions cheaply into the current one.

Workflow content is authored once under `skills/common/`:

- `resume-session.md`
- `close-session.md`
- `review-history.md`

Platform wrappers live under:

- `skills/claude-code/cc-session/SKILL.md`
- `skills/codex/cc-session/SKILL.md`
- `skills/antigravity/cc-session/SKILL.md`

Install selected clients during install:

```bash
./install.sh --clients claude,codex,antigravity
```

```powershell
.\install.ps1 -Clients claude,codex,antigravity
```

No separate `cc-session init` is required.
