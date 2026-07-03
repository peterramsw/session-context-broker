# deterministic-filtering Specification

## Purpose
Compress a raw session transcript without an LLM, retaining conversation
text and every risk/decision signal, so the token-saving core of this tool
works with zero dependency on a Local LLM being configured or available.

## Requirements

### Requirement: Risk and decision signals are retained
The deterministic filter SHALL retain user/assistant conversation text and SHALL preserve evidence for errors, warnings, rollbacks, blockers, corrections, exit codes, git state, test summaries, final reports, and subagent blockers.

#### Scenario: Errors and test summaries survive filtering
- **WHEN** a session contains a failed command, a warning, and a test summary
- **THEN** the filtered output SHALL include structured summaries for those signals

### Requirement: Compressed tools produce evidence summaries
Every compressed or unknown tool call SHALL emit an evidence summary with stable IDs, tool name, argument/result summaries, status, exit code when available, raw/filtered char counts, and source location.

#### Scenario: Unknown tool is still expandable
- **WHEN** an unknown tool event is filtered
- **THEN** the filtered output SHALL include an evidence ID that can be expanded back to the source bytes
