## ADDED Requirements

### Requirement: File-based per-session storage layout
The system SHALL store derived artifacts under `<storage_root>/<provider>/<session-id>/` as `manifest.json`, `normalized.jsonl`, `filtered.jsonl`, `evidence-index.json`, `handoff.json`, and `handoff.md`, with `storage_root` defaulting to `~/.session-context` and overridable via config/`SESSION_CONTEXT_STORAGE_ROOT`.

#### Scenario: Artifacts land in the expected location
- **WHEN** a handoff is created for a Claude Code session with default configuration
- **THEN** its artifacts SHALL be written under `~/.session-context/claude_code/<session-id>/`

### Requirement: Deterministic, non-random evidence IDs
Evidence IDs SHALL be derived deterministically from stable fields (session ID, event ID, evidence kind), not from random UUIDs or insertion order, so the same unmodified session produces the same evidence IDs on every run.

#### Scenario: Rerunning filter produces stable evidence IDs
- **WHEN** `filter_session` is run twice against the same unmodified session
- **THEN** every evidence record's `evidence_id` SHALL be identical across both runs

### Requirement: Evidence expansion by ID with size limits
The system SHALL support expanding one or more evidence records by ID, returning their original content up to a caller-specified maximum size, with an explicit flag indicating whether the returned content was truncated.

#### Scenario: Oversized evidence is truncated and flagged
- **WHEN** `expand_evidence` is called with `max_chars_per_item` smaller than the evidence's raw size
- **THEN** the response SHALL return content truncated to that limit and SHALL set a truncated indicator to true

#### Scenario: Evidence within limit is returned whole
- **WHEN** the requested evidence's raw size is within `max_chars_per_item`
- **THEN** the full content SHALL be returned with the truncated indicator set to false

### Requirement: Path traversal and symlink escape protection
Evidence expansion and any file access derived from a session/evidence reference SHALL reject paths that resolve (after symlink resolution) outside the configured allowed session roots or storage root.

#### Scenario: Traversal attempt is rejected
- **WHEN** a source location or evidence reference resolves to a path outside the allowed roots via `..` segments or a symlink
- **THEN** the system SHALL refuse to read it and SHALL return a clear error rather than the file's contents

### Requirement: Safe concurrent multi-process access
The evidence store SHALL be safe to read and write from multiple concurrent OS processes (e.g. one `cc-session serve-mcp` process each for Claude Code, Codex, and Antigravity running at the same time) without corrupting artifact files or deadlocking.

#### Scenario: Concurrent readers never see a partially written file
- **WHEN** one process is writing `handoff.json` for a session at the same moment another process reads it
- **THEN** the reader SHALL observe either the complete previous version or the complete new version, never a partial/corrupt write

#### Scenario: Concurrent handoff creation for the same session does not corrupt state
- **WHEN** two processes both attempt to create a handoff for the same session at nearly the same time
- **THEN** exactly one SHALL perform the write and the other SHALL either wait briefly and reuse the result or receive a clear "in progress" error, and the resulting `handoff.json` SHALL be well-formed either way

#### Scenario: A slow LLM call in one process does not block other processes' unrelated work
- **WHEN** one process is waiting on a slow/hung Local LLM distiller call for session A
- **THEN** another process performing `list_sessions`, `get_handoff`, or a handoff for a different session B SHALL NOT be blocked by that in-flight call
