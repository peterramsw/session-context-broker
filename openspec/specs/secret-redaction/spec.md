# secret-redaction Specification

## Purpose
Mask credentials and secret-like values at every output/expansion boundary
by default, without ever mutating the raw session on disk, so filtered
artifacts and evidence expansion are safe to hand to a Local LLM or an
agent by default.

## Requirements

### Requirement: Redaction at output boundaries
Filtered artifacts, handoff inputs, and evidence expansion responses SHALL redact credentials and secrets by default without mutating raw session files.

#### Scenario: Raw session remains untouched
- **WHEN** a raw transcript contains a token-like value
- **THEN** filtered and expanded outputs SHALL redact it while the raw transcript on disk remains unchanged

### Requirement: Unredacted access is explicit
Unredacted evidence expansion SHALL require an explicit config/flag override that is disabled by default.

#### Scenario: Default expansion is redacted
- **WHEN** evidence is expanded without the override
- **THEN** secret-like values SHALL be masked
