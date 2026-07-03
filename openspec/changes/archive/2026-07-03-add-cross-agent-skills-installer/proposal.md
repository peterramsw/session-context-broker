## Why

MCP gives agents tools, but it does not define how an agent should resume, close, or review session history. The workflow instructions should live in thin, installable Skills while keeping runtime session processing lightweight.

## What Changes

- Add shared Resume, Close, and Review-History workflow documents.
- Add thin Claude Code, Codex, and Google Antigravity standalone app Skill wrappers.
- Update install scripts to choose target clients during install, showing already-installed integrations as checked.
- Document verified install paths and frontmatter conventions.

## Scope Boundary

This change does not add new parser or evidence-store behavior. It packages the existing/MCP tool surface for agent use.
