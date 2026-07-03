## ADDED Requirements

### Requirement: Stdio MCP server exposing session context tools
The system SHALL provide `cc-session serve-mcp`, reachable over stdio, exposing session listing, inspection, filtering, handoff, search, evidence expansion, context-size comparison, and workspace verification tools. The server SHALL conform to the MCP stdio transport (newline-delimited JSON-RPC) and advertise each tool with a typed input schema.

#### Scenario: Server advertises tools
- **WHEN** an MCP client connects over stdio
- **THEN** it SHALL be able to list the supported tools, each with a typed input schema describing its arguments

#### Scenario: Protocol conformance verified by a real client
- **WHEN** an MCP client performs the initialize → tools/list → tools/call handshake over stdio
- **THEN** every message SHALL be exchanged as newline-delimited JSON-RPC per the MCP stdio transport
- **AND** conformance SHALL be proven by an end-to-end client round-trip that parses each response as JSON-RPC, not by substring matching on self-framed output

### Requirement: MCP and CLI share core logic
The MCP server SHALL call the same internal core packages as the CLI and SHALL NOT duplicate provider/session/handoff logic or shell out to the CLI.

#### Scenario: Bug fix applies to both surfaces
- **WHEN** a core handoff policy bug is fixed
- **THEN** both CLI handoff and MCP `create_handoff` SHALL reflect the fix

### Requirement: Read-only workspace verification
`verify_workspace` SHALL only perform read-only git inspection inside configured allowed roots.

#### Scenario: Outside root is refused
- **WHEN** a workspace path resolves outside `allowed_workspace_roots`
- **THEN** verification SHALL return an error and perform no git operations

### Requirement: Concurrent multi-client processes
Multiple independent stdio server processes SHALL be able to read/write the same evidence store without corrupting artifacts or blocking unrelated reads.

#### Scenario: Three clients coexist
- **WHEN** Claude Code, Codex, and Antigravity each launch their own server process
- **THEN** read-oriented tools SHALL work concurrently against the same store
