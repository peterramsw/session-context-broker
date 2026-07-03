# cli-handoff-commands Specification

## Purpose
TBD - created by archiving change add-codex-llm-handoff. Update Purpose after archive.
## Requirements
### Requirement: Provider-aware CLI slice
The system SHALL add provider-aware support for `list`, `inspect`, `filter`, `stats`, and `handoff` without breaking existing upstream command forms.

#### Scenario: Provider list works
- **WHEN** `cc-session list --provider codex` or `cc-session list --provider antigravity` is run
- **THEN** only sessions from that provider SHALL be listed

#### Scenario: Provider inspect/filter/stats work
- **WHEN** `inspect`, `filter`, or `stats --no-tokens` is run with `--provider codex` or `--provider antigravity`
- **THEN** the command SHALL parse the provider session and report or emit deterministic filtered content

### Requirement: Handoff works without requiring Local LLM
`cc-session handoff` SHALL support `--provider`, `--force`, and `--llm auto|always|never`. It SHALL write a filtered artifact before any Local LLM call and SHALL allow filtered-only handoff when Local LLM is absent or disabled by policy.

#### Scenario: Auto mode falls back to filtered output
- **WHEN** `cc-session handoff --llm auto <session>` is run for a small filtered transcript or without a configured Local LLM endpoint
- **THEN** the command SHALL write `filtered.md` and SHALL NOT fail solely because Local LLM is unavailable

#### Scenario: Explicit Local LLM request fails loud
- **WHEN** `cc-session handoff --llm always <session>` is run without a configured Local LLM endpoint
- **THEN** the command SHALL write `filtered.md` first and then return an actionable Local LLM configuration error

### Requirement: Remaining CLI commands backed by shared core
The system SHALL add `search`, evidence expansion, `verify-workspace`, and `serve-mcp` commands without breaking upstream `read`, `context`, `stats`, `audit`, `expand`, or `inject`.

#### Scenario: Existing commands are unaffected
- **WHEN** the upstream CLI test suite is run
- **THEN** previously-passing tests SHALL still pass without expected-output changes

