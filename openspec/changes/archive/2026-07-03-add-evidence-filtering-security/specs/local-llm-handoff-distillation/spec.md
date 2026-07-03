## ADDED Requirements

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
