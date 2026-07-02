## ADDED Requirements

### Requirement: Remaining CLI commands backed by shared core
The system SHALL add `search`, evidence expansion, `verify-workspace`, and `serve-mcp` commands without breaking upstream `read`, `context`, `stats`, `audit`, `expand`, or `inject`.

#### Scenario: Existing commands are unaffected
- **WHEN** the upstream CLI test suite is run
- **THEN** previously-passing tests SHALL still pass without expected-output changes
