## ADDED Requirements

### Requirement: Single source of shared workflow content
The Resume, Close, and Review History workflow instructions SHALL be authored once in `skills/common/{resume-session.md,close-session.md,review-history.md}`. Per-platform skill files SHALL reference/include this shared content rather than duplicating it.

#### Scenario: Editing shared content updates all platforms
- **WHEN** `skills/common/resume-session.md` is edited
- **THEN** the Claude Code, Codex, and Antigravity skill packages SHALL all reflect the change without a separate edit to any platform file

### Requirement: Claude Code, Codex, and Antigravity installable skills
The system SHALL provide `skills/claude-code/SKILL.md`, `skills/codex/SKILL.md`, and `skills/antigravity/SKILL.md`, each installable via that platform's normal skill-installation convention, each a thin wrapper that points to the shared `skills/common/` content plus whatever platform-specific frontmatter/invocation format that platform requires. Antigravity 2.0 (the standalone conversational multi-agent desktop app, distinct from the older Antigravity IDE) SHALL be treated as a first-class Skill target, not merely an MCP client, since it has its own Skill convention (`~/.gemini/skills/<name>/SKILL.md`, global install, `tools` frontmatter field referencing MCP tool names).

#### Scenario: Claude Code skill installs and activates
- **WHEN** `skills/claude-code/SKILL.md` is installed to the Claude Code skills directory (`~/.claude/skills/`)
- **THEN** it SHALL be invocable and SHALL surface the Resume/Close/Review-History workflows

#### Scenario: Antigravity skill installs and activates
- **WHEN** `skills/antigravity/SKILL.md` is installed to Antigravity's global skills directory (`~/.gemini/skills/`)
- **THEN** it SHALL be selectable by Antigravity and SHALL surface the Resume/Close/Review-History workflows, with its frontmatter correctly scoping the `tools` it needs from the session-context MCP server

#### Scenario: Same filename, different platforms, no collision
- **WHEN** both a Claude Code skill and an Antigravity skill are installed on the same machine
- **THEN** each SHALL live in its own platform-specific directory (`~/.claude/skills/...` vs `~/.gemini/skills/...`) so the shared `SKILL.md` filename never causes one platform's install to overwrite the other's

### Requirement: Resume workflow never trusts unverified claims
The Resume workflow SHALL: find the prior session, load or create its handoff, present the prior objective/confirmed items/open items/blockers/next steps to the user, then call `verify_workspace` and compare its repository/branch/commit/uncommitted-changes state against the handoff before treating any of the handoff's completion claims (tests passed, deployment completed, migration applied, rollback completed) as true. If current work depends on such a claim, the workflow SHALL expand the relevant evidence or prompt re-verification before proceeding, and it SHALL NOT re-adopt an option already recorded as rejected or superseded.

#### Scenario: A stale "tests passed" claim is not trusted blindly
- **WHEN** the handoff states a test suite passed, but the current work depends on that being true
- **THEN** the Resume workflow SHALL expand the underlying evidence or prompt re-running verification before proceeding, rather than proceeding solely on the handoff's word

#### Scenario: A rejected approach is not silently retried
- **WHEN** the handoff records an approach as rejected/superseded
- **THEN** the Resume workflow SHALL NOT propose re-adopting that approach without the user explicitly revisiting it

### Requirement: Close workflow produces and validates a fresh handoff
The Close workflow SHALL gather the current repository/branch/commit/git-status, summarize this session's changes/tests/warnings/blockers/deployment-or-rollback activity, run `filter_session` and `create_handoff`, surface any validation warnings/conflicts, and report handoff success/failure with raw/filtered/handoff token estimates and unverified-claims, without silently ignoring conflicts.

#### Scenario: A validation conflict blocks a silent success report
- **WHEN** `create_handoff` reports a validation conflict (e.g. deployment/rollback conflict)
- **THEN** the Close workflow SHALL surface that conflict to the user rather than reporting the close as a clean success

### Requirement: Review History workflow only proposes candidates
The Review History workflow SHALL read one or more session handoffs, identify repeated user corrections, repeated tool misuse, commonly-skipped verification, recurring blockers, and candidates suitable for a skill/hook/AGENTS.md/CLAUDE.md change — each candidate accompanied by a session/evidence reference — and SHALL NOT itself modify any rule file (AGENTS.md, CLAUDE.md, hooks, or skills).

#### Scenario: A recurring correction becomes a reported candidate, not an auto-edit
- **WHEN** the same user correction appears across multiple past sessions
- **THEN** the workflow SHALL report it as a candidate improvement with supporting evidence references, and SHALL NOT edit any AGENTS.md/CLAUDE.md file itself
