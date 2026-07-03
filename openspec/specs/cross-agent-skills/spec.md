# cross-agent-skills Specification

## Purpose
Give Claude Code, Codex, and Google Antigravity thin, installable Skills
that teach an agent how to resume, close, and review sessions using the
session-context broker's CLI/MCP surface, without duplicating workflow
content per platform.

## Requirements

### Requirement: Single source of workflow content
Resume, Close, and Review-History instructions SHALL be authored once under `skills/common/` and referenced by platform wrappers.

#### Scenario: Shared content updates all wrappers
- **WHEN** common resume workflow text changes
- **THEN** Claude Code, Codex, and Antigravity wrappers SHALL use the updated content without duplicating it

### Requirement: Installable cross-agent skills
The system SHALL provide installable Skill wrappers for Claude Code, Codex, and Google Antigravity standalone app, each using that platform's normal convention.

#### Scenario: Same filename does not collide
- **WHEN** Claude Code and Antigravity both require a file named `SKILL.md`
- **THEN** each SHALL be installed in its platform-specific directory

### Requirement: Installer selects client integrations during install
The installer SHALL offer Claude Code, Codex, and Antigravity targets during install, showing already-installed targets as checked by default and supporting non-interactive client selection.

#### Scenario: No separate init is required
- **WHEN** installation finishes with selected client integrations
- **THEN** those client integrations SHALL be ready without running a separate `cc-session init`

### Requirement: Resume workflow verifies claims
The Resume workflow SHALL not trust handoff claims about tests, deployment, rollback, branch, or commit without evidence or re-verification.

#### Scenario: Stale tests-passed claim is not trusted
- **WHEN** a handoff says tests passed
- **THEN** Resume SHALL expand evidence or prompt re-verification before relying on it

### Requirement: Review History only proposes candidates
Review History SHALL report candidates for rule/skill/hook improvements with evidence and SHALL NOT edit rule files itself.

#### Scenario: Repeated correction is reported, not auto-applied
- **WHEN** a repeated user correction is found
- **THEN** the workflow SHALL report it with evidence and leave edits to the user
