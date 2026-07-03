## ADDED Requirements

### Requirement: Config is documented and testable
Configuration for session sources, storage root, allowed roots, and optional Local LLM settings SHALL be documented with file and environment variable examples.

#### Scenario: User can configure without reading source
- **WHEN** a user reads the README and config docs
- **THEN** they SHALL be able to configure Claude Code, Codex, Antigravity, and optional Local LLM endpoints

### Requirement: Documentation covers release behavior
The documentation SHALL cover architecture, provider formats, normalized events, handoff schema, Local LLM behavior, MCP tools, Skills, security, and upstream sync.

#### Scenario: Provider docs cite real targets
- **WHEN** provider docs describe Antigravity
- **THEN** they SHALL identify the Google standalone app brain store separately from Antigravity IDE storage

### Requirement: End-to-end validation is reproducible
The project SHALL include a full pipeline e2e test and an opt-in live Local LLM integration test skipped by default.

#### Scenario: Default tests stay offline
- **WHEN** `go test ./...` is run without live-test env vars
- **THEN** no Local LLM endpoint SHALL be required
