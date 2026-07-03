# local-llm-handoff-distillation Specification

## Purpose
TBD - created by archiving change add-codex-llm-handoff. Update Purpose after archive.
## Requirements
### Requirement: Optional configurable OpenAI-compatible endpoint
Local LLM handoff generation SHALL be disabled unless explicitly configured. When enabled, it SHALL call a locally configured OpenAI-compatible chat completions endpoint using configured base URL, API key, model, timeout, max output tokens, and sampling settings.

#### Scenario: Empty API key is accepted
- **WHEN** Local LLM handoff generation is enabled and `local_llm.api_key` is empty
- **THEN** the client SHALL omit the authorization header and still call the local endpoint

#### Scenario: Deterministic sampling is sent
- **WHEN** Local LLM handoff generation is invoked
- **THEN** the request SHALL include `temperature` with default `0` and SHALL include configured `top_p`/`top_k` only when configured

### Requirement: Filtered-first Local LLM policy
The system SHALL compute and persist the deterministic redacted filtered transcript before deciding whether to call the Local LLM. Automatic mode SHALL compare redacted filtered transcript size against `local_llm.min_filtered_chars`.

#### Scenario: Short session skips Local LLM automatically
- **WHEN** `handoff --llm auto` is run for a session below the threshold
- **THEN** the system SHALL write `filtered.md` and SHALL NOT call the Local LLM endpoint

#### Scenario: Large session uses Local LLM when configured
- **WHEN** `handoff --llm auto` is run for a session above the threshold and Local LLM is configured
- **THEN** the system SHALL call the Local LLM and write `handoff.json` and `handoff.md`

### Requirement: Handoff schema and disclosure
Generated handoff artifacts SHALL include the handoff schema version, session metadata, objective/state/action sections, validation fields, and a rendered Markdown disclosure stating that the handoff is derived and not the source of truth.

#### Scenario: Disclosure is present
- **WHEN** `handoff.md` is rendered
- **THEN** it SHALL contain the derived-artifact disclosure statement

### Requirement: One repair attempt, then fail loud
If the Local LLM output is malformed JSON, the system SHALL attempt one repair request. If repair fails, it SHALL preserve the raw failed output and return an error rather than writing an empty handoff.

#### Scenario: Repair succeeds
- **WHEN** the first response is malformed and the repair response is valid
- **THEN** the repaired handoff SHALL be written

### Requirement: Evidence-backed handoff validation
Confirmed decisions, test results, deployment/rollback claims, blockers, and user corrections SHALL include resolvable evidence references or be demoted to claims requiring re-verification.

#### Scenario: Unevidenced tests passed claim is demoted
- **WHEN** a handoff claims tests passed without supporting evidence
- **THEN** the claim SHALL appear under `claims_requiring_reverification`

### Requirement: Chunked distillation preserves evidence
Oversized filtered transcripts SHALL be chunked without reordering events, and merged partial handoffs SHALL drop unresolved evidence references with warnings.

#### Scenario: Merge does not invent evidence
- **WHEN** a chunk result references an evidence ID that does not exist
- **THEN** the merged handoff SHALL omit that reference and record a warning

