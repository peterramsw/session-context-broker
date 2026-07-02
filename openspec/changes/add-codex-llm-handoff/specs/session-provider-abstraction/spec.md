## ADDED Requirements

### Requirement: Normalized provider event schema
The system SHALL define normalized session provider types including `SessionRef`, `SessionMetadata`, `SessionEvent`, `SessionTool`, `EventSource`, and deterministic event ID derivation.

#### Scenario: Stable event IDs are reproducible
- **WHEN** an unmodified Codex or Antigravity session is parsed twice
- **THEN** every event ID SHALL remain stable across parses

### Requirement: Codex session adapter
The system SHALL provide a Codex adapter that discovers, inspects, parses, and reads local Codex session JSONL files into normalized events and the existing analyzer event model.

#### Scenario: Codex fixture parses
- **WHEN** the Codex fixture is parsed
- **THEN** user messages, assistant messages, tool calls/results, reasoning/noise, malformed lines, and unknown events SHALL be handled without dropping valid events

### Requirement: Google Antigravity standalone app adapter
The system SHALL provide an Antigravity adapter for Google's standalone Antigravity app brain store, not Antigravity IDE storage. On Windows, the default root SHALL include `~/.gemini/antigravity/brain`, and sessions SHALL be discovered from `<conversation-id>/.system_generated/logs/transcript_full.jsonl` or `transcript.jsonl`, preferring `transcript_full.jsonl` when present.

#### Scenario: Standalone brain session is parsed
- **WHEN** an Antigravity standalone conversation exists under `~/.gemini/antigravity/brain/<conversation-id>/.system_generated/logs/`
- **THEN** provider-aware `list`, `inspect`, `filter`, `stats`, and `handoff` commands SHALL resolve that conversation and SHALL NOT read from Antigravity IDE storage

#### Scenario: Antigravity step events normalize into shared events
- **WHEN** an Antigravity transcript contains `USER_INPUT`, `PLANNER_RESPONSE`, tool-call entries, and tool execution result steps
- **THEN** the adapter SHALL normalize them into shared events as user/assistant messages, tool calls, tool results, reasoning/noise, or parse errors while preserving source location
