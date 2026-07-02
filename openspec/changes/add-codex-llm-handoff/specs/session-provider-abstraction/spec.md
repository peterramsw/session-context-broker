## ADDED Requirements

### Requirement: SessionProvider interface
The system SHALL define a `SessionProvider` interface (`Name`, `Discover`, `Inspect`, `Parse`) in `internal/session`, and all provider-specific parsing logic SHALL implement this interface rather than being referenced by conditional branches in shared code.

#### Scenario: Core packages never branch on provider type
- **WHEN** `internal/filtering`, `internal/evidence`, `internal/handoff`, or `internal/mcp` need to process a session
- **THEN** they SHALL depend only on `internal/session` types (`SessionProvider`, `SessionEvent`, `SessionMetadata`, `SessionRef`) and SHALL NOT import `internal/claudecodec` or `internal/codexcodec` directly

### Requirement: Normalized cross-provider event schema
The system SHALL normalize every parsed event from any provider into a single `SessionEvent` shape containing `event_id`, `session_id`, `provider`, `timestamp`, `sequence`, `role`, `event_type`, `content`, a `tool` sub-object (`call_id`, `name`, `arguments`, `result`, `stdout`, `stderr`, `exit_code`, `status`, `duration_ms`), a `source` sub-object (`path`, `line_start`, `line_end`, `byte_start`, `byte_end`, `content_hash`), and `metadata`.

#### Scenario: Claude Code, Codex, and Antigravity events share one schema
- **WHEN** Claude Code, Codex, and Antigravity sessions are parsed
- **THEN** all produce `SessionEvent` values of the same Go type, consumable by the same downstream filtering/evidence/handoff code without type switches

### Requirement: Deterministic, stable event IDs
Event IDs SHALL be derived deterministically from stable identifying fields (session ID, provider, sequence, source location, event type) rather than randomly generated, so that re-parsing an unmodified session produces identical event IDs.

#### Scenario: Rerun produces identical IDs
- **WHEN** the same unmodified session file is parsed twice
- **THEN** every event's `event_id` SHALL be identical across both parses

### Requirement: Source traceability
Every normalized event SHALL retain enough source location information (`path`, `line_start`/`line_end` or `byte_start`/`byte_end`, `content_hash`) to re-locate and re-read the exact original bytes it was derived from.

#### Scenario: Evidence expansion re-reads original bytes
- **WHEN** a downstream tool requests the raw content behind an event
- **THEN** the system SHALL use the event's `source` fields to read the corresponding byte range from the original session file, not from a cached copy

### Requirement: Graceful handling of malformed and unknown input
A malformed line in a session file SHALL NOT abort parsing of the rest of that session; it SHALL be recorded as a parse error against its position and parsing SHALL continue. An event of a type the parser does not recognize SHALL be normalized with `event_type: "unknown"` and its raw content preserved, never silently dropped.

#### Scenario: One malformed line does not lose the session
- **WHEN** a session file contains one corrupted/truncated JSON line among many valid lines
- **THEN** the system SHALL successfully parse all valid lines, SHALL surface the malformed line as a recorded parse error with its location, and SHALL NOT fail the entire parse

#### Scenario: Unknown event type is preserved
- **WHEN** a session contains an event type not explicitly modeled by the parser
- **THEN** the system SHALL emit a normalized event with `event_type: "unknown"`, preserving the original content and source location, rather than omitting it

### Requirement: Claude Code adapter behavioral compatibility
The existing Claude Code parsing behavior (via `internal/claudecodec`) SHALL be preserved exactly when accessed through the new `SessionProvider` interface — existing upstream tests SHALL continue to pass unmodified.

#### Scenario: Upstream Claude Code tests still pass
- **WHEN** the existing upstream test suite is run after the `SessionProvider` refactor
- **THEN** all previously-passing tests SHALL continue to pass with no behavioral changes

### Requirement: Codex session adapter
The system SHALL provide a `internal/codexcodec` adapter implementing `SessionProvider` for Codex CLI sessions, built against the real local Codex session format discovered during implementation (or a clearly-labeled synthetic fixture if no real Codex session is available on the development machine), covering user/assistant turns, tool calls/results, reasoning events, and compaction events.

#### Scenario: Real or synthetic Codex fixture is documented
- **WHEN** the Codex adapter is implemented
- **THEN** `docs/session-provider.md` SHALL record which Codex session schema/version was targeted, and whether the test fixture is real (redacted) or synthetic

#### Scenario: Unknown Codex event format degrades gracefully
- **WHEN** the Codex adapter encounters an event shape it does not recognize
- **THEN** it SHALL normalize it as `event_type: "unknown"` with preserved content, consistent with the malformed/unknown-input requirement above, rather than failing the whole parse

### Requirement: Antigravity session adapter target
The system SHALL treat Antigravity as a first-class provider target alongside Claude Code and Codex. If the local Antigravity session format is not yet verified, the provider SHALL fail with an actionable "format not implemented yet" error rather than being omitted from configuration, documentation, or user-facing provider lists.

#### Scenario: Antigravity is visible even before parsing ships
- **WHEN** a user lists supported providers
- **THEN** `antigravity` SHALL appear as a recognized provider target, and any unavailable local parsing support SHALL be reported as a clear implementation status rather than as an unknown provider

### Requirement: Google Antigravity standalone app adapter
The Antigravity adapter SHALL target Google's standalone Antigravity app local brain store, not the Antigravity IDE's VS Code-style storage. On Windows, the default root SHALL include `~/.gemini/antigravity/brain`, and sessions SHALL be discovered from `<conversation-id>/.system_generated/logs/transcript_full.jsonl` or `transcript.jsonl`, preferring `transcript_full.jsonl` when present.

#### Scenario: Standalone brain session is parsed
- **WHEN** an Antigravity standalone conversation exists under `~/.gemini/antigravity/brain/<conversation-id>/.system_generated/logs/`
- **THEN** `cc-session list --provider antigravity`, `inspect --provider antigravity`, `filter --provider antigravity`, `stats --provider antigravity`, and `handoff --provider antigravity` SHALL resolve that conversation and SHALL NOT read from the Antigravity IDE storage path

#### Scenario: Antigravity step events normalize into shared events
- **WHEN** an Antigravity transcript contains `USER_INPUT`, `PLANNER_RESPONSE`, tool-call entries, and tool execution result steps such as `RUN_COMMAND`, `VIEW_FILE`, `MCP_TOOL`, or `ERROR_MESSAGE`
- **THEN** the adapter SHALL normalize them into the shared `SessionEvent` schema as user/assistant messages, tool calls, tool results, or noise while preserving source location and deterministic event IDs

### Requirement: Session discovery across platforms and overrides
The system SHALL discover Claude Code, Codex, and Antigravity session roots by checking, in order: explicit config file entries, environment variable overrides (`CLAUDE_SESSION_ROOTS`, `CODEX_SESSION_ROOTS`, `ANTIGRAVITY_SESSION_ROOTS`), and common platform-default locations for Windows and Linux. If no sources can be found, the system SHALL return a clear, actionable error describing how to configure a root.

#### Scenario: Environment variable override takes effect
- **WHEN** `CODEX_SESSION_ROOTS` is set to a specific directory
- **THEN** session discovery for the Codex provider SHALL use that directory instead of the platform default

#### Scenario: No sources found yields an actionable error
- **WHEN** no config, env var, or platform-default root contains any session data for a provider
- **THEN** the system SHALL return an error message that names the provider and explains how to configure `session_sources.<provider>.roots` or the corresponding environment variable
