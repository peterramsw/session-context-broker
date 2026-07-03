## ADDED Requirements

### Requirement: File-based evidence store
The system SHALL write per-session artifacts under `storage_root/<provider>/<session-id>/`, including manifest, normalized events, filtered transcript, evidence index, and handoff artifacts.

#### Scenario: Evidence store is reproducible
- **WHEN** the same unmodified session is processed twice
- **THEN** deterministic event and evidence IDs SHALL remain stable

### Requirement: Safe evidence expansion
Evidence expansion SHALL re-read allowed source bytes by evidence ID, enforce size limits, report truncation, and reject path traversal or symlink escape.

#### Scenario: Traversal is refused
- **WHEN** an evidence lookup would read outside configured session roots
- **THEN** the read SHALL be rejected

### Requirement: Concurrent local writers are safe
Artifact writes SHALL use atomic temp-file-then-rename writes and short-lived per-session advisory locks only around critical write sections.

#### Scenario: Slow LLM call does not block unrelated reads
- **WHEN** one process is waiting on a Local LLM call
- **THEN** other processes SHALL still read unrelated sessions from the evidence store
