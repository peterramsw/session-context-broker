## ADDED Requirements

### Requirement: Optional configurable OpenAI-compatible endpoint
The distiller SHALL be disabled unless Local LLM handoff generation is explicitly enabled. When enabled, it SHALL call a locally-configured OpenAI-compatible chat completions endpoint, with base URL, API key (optionally empty), model name, and timeout all read from configuration/environment — never hardcoded.

#### Scenario: No endpoint still supports deterministic workflows
- **WHEN** `local_llm.enabled` is false or `local_llm.base_url` is unset
- **THEN** deterministic `list`, `inspect`, `filter`, `search`, `expand`, and context-size comparison workflows SHALL still succeed without making any Local LLM request

#### Scenario: Empty API key is accepted for an unauthenticated local endpoint
- **WHEN** Local LLM handoff generation is enabled and `local_llm.api_key` is empty
- **THEN** the distiller SHALL still make requests to `local_llm.base_url` successfully, omitting or emptying the auth header rather than failing

### Requirement: Filtered-first Local LLM policy
The system SHALL always compute and persist the deterministic filtered transcript before deciding whether to call the Local LLM. The automatic decision SHALL use the redacted filtered transcript size, not the raw session size, against a configurable `local_llm.min_filtered_chars` threshold. The default threshold SHALL skip short sessions and preserve filtered-only operation for users who do not configure a Local LLM.

#### Scenario: Short session skips Local LLM automatically
- **WHEN** `handoff --llm auto` is run for a session whose redacted filtered transcript is below `local_llm.min_filtered_chars`
- **THEN** the system SHALL write `filtered.md` and SHALL NOT call the Local LLM endpoint

#### Scenario: Large session can still work without a Local LLM
- **WHEN** `handoff --llm auto` is run for a session above `local_llm.min_filtered_chars` but Local LLM is disabled or unconfigured
- **THEN** the system SHALL still write `filtered.md`, SHALL report that Local LLM was skipped because it is unavailable, and SHALL NOT fail solely because the optional Local LLM is absent

#### Scenario: Explicit Local LLM request fails loud
- **WHEN** `handoff --llm always` is run and Local LLM is disabled or missing `base_url`/`model`
- **THEN** the system SHALL write `filtered.md` first and then return an actionable Local LLM configuration error

### Requirement: Restricted distiller input
The distiller's prompt SHALL consist only of the filtered transcript, evidence metadata, the handoff JSON schema, and explicit extraction rules. It SHALL NOT include the unfiltered raw session unless the caller explicitly enables a debug mode.

#### Scenario: Raw session is excluded by default
- **WHEN** a handoff is generated without debug mode enabled
- **THEN** the request sent to the local LLM endpoint SHALL NOT contain the raw/unfiltered session content

### Requirement: Handoff schema conformance
The distiller SHALL produce JSON conforming to the handoff schema, including `schema_version`, `session`, `objective`, `confirmed_decisions`, `rejected_or_superseded`, `implementation_state`, `verification` (`passed`/`failed`/`not_run`/`warnings`), `deployment`, `known_blockers`, `unresolved_questions`, `next_actions`, `user_corrections`, `claims_requiring_reverification`, `workflow_improvement_candidates`, and `validation` (`warnings`/`conflicts`).

#### Scenario: A generated handoff includes all top-level sections
- **WHEN** `create_handoff` completes successfully
- **THEN** the resulting `handoff.json` SHALL contain all of the top-level fields listed above, using empty arrays/objects where a section has no content rather than omitting the key

### Requirement: Evidence-backed claims
`confirmed_decisions`, `verification.passed`, `verification.failed`, `deployment`, rollback-related entries, `known_blockers`, and `user_corrections` entries SHALL include `evidence_refs` wherever the underlying transcript supports it. A claim in one of these categories with no supporting evidence SHALL NOT be emitted as confirmed; it SHALL instead appear under `claims_requiring_reverification`.

#### Scenario: Unevidenced "tests passed" is demoted
- **WHEN** the filtered transcript contains no evidence-backed test-passing signal for a claim the model wants to make
- **THEN** that claim SHALL appear in `claims_requiring_reverification`, not in `verification.passed`

### Requirement: Handoff schema and evidence validation
The system SHALL validate every produced handoff both structurally (schema conformance) and semantically: every `evidence_refs` entry SHALL resolve to a real evidence ID or be rejected with a warning; a `deployment.completed` alongside rollback evidence SHALL raise a conflict; a handoff's recorded branch/commit that mismatches the session's recorded branch/commit SHALL raise a warning.

#### Scenario: Hallucinated evidence ID is caught
- **WHEN** the distiller's output references an `evidence_ref` that does not exist in the evidence index
- **THEN** validation SHALL reject or strip that reference and record a warning, and it SHALL NOT be silently accepted as valid

#### Scenario: Deployment/rollback conflict is flagged
- **WHEN** a handoff states `deployment.completed = true` and also includes rollback evidence for the same deployment
- **THEN** validation SHALL record a conflict in `validation.conflicts`

### Requirement: One repair attempt, then fail loud
If the distiller's raw output is not valid JSON, the system SHALL attempt exactly one automated repair round-trip. If the repaired output still fails schema validation, the system SHALL preserve the raw failed output on disk and return an error — it SHALL NOT silently produce an empty or placeholder handoff.

#### Scenario: Repair succeeds
- **WHEN** the first distiller response is malformed JSON but a repair re-prompt yields valid, schema-conforming JSON
- **THEN** the repaired result SHALL be used as the handoff

#### Scenario: Repair fails and the raw output is preserved
- **WHEN** both the original and repaired distiller responses fail validation
- **THEN** the system SHALL write the raw failed output to disk alongside where `handoff.json` would have gone, and SHALL return an error to the caller rather than writing an empty handoff

### Requirement: Context budgeting via phase-based chunking
When the filtered transcript exceeds the configured context budget, the system SHALL split it into ordered chunks along heuristic phase boundaries (e.g. requirement/planning/design/implementation/debugging/testing/deployment/rollback/final-report) without reordering events, distill each chunk independently, and merge the partial results while preserving evidence references and event order.

#### Scenario: Oversized transcript is chunked, not truncated
- **WHEN** a filtered transcript exceeds `local_llm.max_context`
- **THEN** the system SHALL chunk it rather than dropping its earliest or latest content outright

#### Scenario: Merge does not invent evidence
- **WHEN** partial per-chunk results are merged into a final handoff
- **THEN** any evidence reference in a partial result that does not exist in the evidence index SHALL be dropped from the merged result with a warning, not carried through

### Requirement: Derived-artifact disclosure
Every rendered `handoff.md` SHALL contain, verbatim, the statement: "This handoff is a derived artifact and must not be treated as the source of truth."

#### Scenario: Disclosure is present in every handoff
- **WHEN** `handoff.md` is rendered for any session
- **THEN** it SHALL contain that exact sentence
