## ADDED Requirements

### Requirement: No-LLM first-pass compression
The first-pass compression of a normalized session into a filtered transcript SHALL be entirely deterministic (no LLM call), extending the existing `internal/summarizer`/`internal/analyzer` logic rather than replacing it.

#### Scenario: Filtering works with no Qwen endpoint configured
- **WHEN** `qwen.base_url` is unset or unreachable
- **THEN** `cc-session filter`/`filter_session` SHALL still successfully produce a filtered transcript

### Requirement: Guaranteed retention of risk and decision signals
The deterministic filter SHALL fully or near-fully retain: user prompts, assistant natural-language replies, user corrections to the agent, requirement changes, explicit decisions, explicit rejections, and any occurrence of error, warning, failed, timeout, permission-denied, rollback, blocked, skipped, not-run, or partial-success signals, tool exit codes, test summaries, git branch/commit/status/diff summaries, migration/deployment/rollback results, final agent reports, and blockers/failures reported by subagents.

#### Scenario: A failed test is never compressed away
- **WHEN** a session contains a tool result indicating a failed test with a stack trace
- **THEN** the filtered transcript SHALL retain the failure indication, the test name, and enough of the failure detail to be actionable, and it SHALL be reachable via evidence reference for full detail

#### Scenario: A user correction is preserved verbatim
- **WHEN** a user message corrects or reverses a prior agent decision
- **THEN** that user message's text SHALL appear in the filtered transcript essentially verbatim, not summarized away

### Requirement: Structured evidence summary for compressed tool calls
Every tool call whose full input/output is compressed out of the filtered transcript SHALL be replaced with a structured summary containing at minimum `evidence_id`, `event_id`, `tool_name`, `arguments_summary`, `result_summary`, `status`, `exit_code`, `warning_count`, `error_count`, `raw_chars`, `filtered_chars`, and `source_location`. A bare string such as "tool succeeded" with no further structure SHALL NOT be considered a valid compressed representation.

#### Scenario: Large successful Bash output is compressed but structured
- **WHEN** a `Bash` tool call produces 50KB of clean, non-error stdout
- **THEN** the filtered transcript SHALL replace the raw stdout with a structured summary record including at least the fields above, and the summary SHALL be expandable back to the original 50KB via its `evidence_id`

### Requirement: Unknown tools are summarized, not dropped
A tool call for a tool name the filter does not have specific handling for SHALL still produce a structured summary (tool name, argument summary, status, result summary, evidence reference) rather than being silently omitted.

#### Scenario: Unrecognized tool still yields a queryable record
- **WHEN** a session contains a call to a tool the filter has no specific rule for
- **THEN** the filtered transcript SHALL contain an evidence-referenced summary record for that call, with its real tool name and a best-effort argument/result summary

### Requirement: Compressible-but-expandable content categories
Large Bash/Read/Grep output, large file contents, repeated system reminders, full skill-injection text, thinking/reasoning content, full subagent output, large JSON blobs, exact duplicate content, and clean/no-anomaly build logs SHALL be eligible for compression, but SHALL always remain expandable by evidence ID without any loss versus the original.

#### Scenario: Repeated system reminder is deduplicated but recoverable
- **WHEN** the same system-reminder text appears many times across a session
- **THEN** the filtered transcript SHALL avoid repeating the full text every time, while each occurrence SHALL remain individually expandable to its exact original text via its own evidence ID

### Requirement: Raw session immutability
No filtering operation SHALL modify, move, or delete the original raw session file.

#### Scenario: Filtering a session does not touch the source file
- **WHEN** `cc-session filter <session>` is run
- **THEN** the raw session file's mtime, size, and content SHALL be unchanged after the command completes
