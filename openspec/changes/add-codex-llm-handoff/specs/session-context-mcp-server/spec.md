## ADDED Requirements

### Requirement: Stdio MCP server exposing the full tool set
The system SHALL provide an MCP server, reachable over stdio, exposing the tools `list_sessions`, `inspect_session`, `filter_session`, `create_handoff`, `get_handoff`, `search_session`, `expand_evidence`, `compare_context_size`, and `verify_workspace`, using a maintained Go MCP SDK rather than a hand-rolled protocol implementation.

#### Scenario: Server starts and advertises its tools
- **WHEN** `cc-session serve-mcp` is launched
- **THEN** an MCP client connecting over stdio SHALL be able to list all nine tools above with their input schemas

### Requirement: Same core logic as the CLI
The MCP server SHALL call the same internal packages (`internal/session`, `internal/filtering`, `internal/evidence`, `internal/handoff`, `internal/workspace`) that the CLI subcommands call directly, rather than duplicating logic or shelling out to the CLI binary.

#### Scenario: CLI and MCP agree on results
- **WHEN** `cc-session stats <session>` and the `inspect_session` MCP tool are run against the same session
- **THEN** their token/character counts and metadata SHALL match

### Requirement: Read-only, sandboxed workspace verification
`verify_workspace` SHALL only ever perform read-only git inspection (status, current branch, current commit, diff stat, untracked files) against a workspace path that resolves inside `allowed_workspace_roots`. It SHALL NOT modify files, commit, reset, checkout, run arbitrary shell commands, deploy, or run migrations.

#### Scenario: Workspace outside allowed roots is rejected
- **WHEN** `verify_workspace` is called with a `workspace_path` that does not resolve inside any configured `allowed_workspace_roots` entry
- **THEN** the tool SHALL return an error and SHALL perform no git operations

#### Scenario: Verification never mutates the workspace
- **WHEN** `verify_workspace` is called against a valid, allowed workspace
- **THEN** no files in that workspace SHALL be modified and no git ref SHALL change as a result

### Requirement: No estimation presented as exact, no pricing
`compare_context_size` and any other token-count output SHALL be clearly labeled as an estimate and SHALL NOT present hardcoded per-provider dollar costs.

#### Scenario: Token counts are labeled as estimates
- **WHEN** `compare_context_size` returns raw/filtered/handoff token counts
- **THEN** the response SHALL indicate these are estimated values, not exact tokenizer output guaranteed to match any specific provider's billing

### Requirement: Evidence and workspace access confined to allowed roots
The MCP server SHALL NOT read any file outside the configured session roots, `storage_root`, and `allowed_workspace_roots`, regardless of what path a tool call requests.

#### Scenario: Arbitrary file read is refused
- **WHEN** a tool call's parameters would cause a file read outside all configured allowed roots
- **THEN** the server SHALL refuse the read and return an error

### Requirement: Concurrent multi-client-process support
The MCP server SHALL support being launched as multiple independent stdio processes simultaneously — one each for Claude Code, Codex, and Antigravity — against the same evidence store, without requiring the client applications to coordinate with each other, without deadlocking, and without one process's slow operation blocking another process's unrelated operations. No container/daemon orchestration is required to satisfy this.

#### Scenario: Three concurrent server processes coexist
- **WHEN** three `cc-session serve-mcp` processes are started at the same time, each connected to a different client app
- **THEN** all three SHALL be able to independently call read-oriented tools (`list_sessions`, `get_handoff`, `search_session`, `expand_evidence`) against the same evidence store without errors attributable to concurrency

#### Scenario: One app can rescue a session another app got stuck on
- **WHEN** one app's process is unresponsive or its session is stuck mid-task
- **THEN** a different concurrently-running app's MCP process SHALL be able to call `get_handoff`/`inspect_session`/`expand_evidence` for that same session and receive correct, uncorrupted results
