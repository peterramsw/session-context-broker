# Skills

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
