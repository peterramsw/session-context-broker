## ADDED Requirements

### Requirement: Deterministic secret detection
The system SHALL detect, via deterministic pattern matching (not an LLM call), at minimum: API keys, bearer tokens, `Authorization` header values, access tokens, refresh tokens, passwords, cookies, client secrets, private keys, common `.env`-style sensitive assignments, GitHub tokens, Anthropic/OpenAI-shaped API tokens, JWTs, and PEM blocks.

#### Scenario: A bearer token in a tool result is detected
- **WHEN** a tool result contains a string matching an `Authorization: Bearer <token>` pattern
- **THEN** the redaction pass SHALL identify and mask that token

### Requirement: Redaction on by default everywhere secrets could leak
Redaction SHALL be applied, by default, before any content is sent to the local LLM distiller, and to all filtered-transcript, handoff, and evidence-expansion output. Unredacted output SHALL only be produced when explicitly enabled via configuration.

#### Scenario: Distiller never sees an unredacted secret by default
- **WHEN** a filtered transcript containing a detected secret is prepared for the Qwen distiller
- **THEN** the secret SHALL already be masked before the distiller request is constructed

#### Scenario: Evidence expand masks secrets by default
- **WHEN** `expand_evidence`/`cc-session expand` is called without an explicit unredacted override
- **THEN** any detected secret in the expanded content SHALL be masked

#### Scenario: Explicit opt-in reveals unredacted content
- **WHEN** the user explicitly configures/passes an unredacted override
- **THEN** the system SHALL return unredacted content for that call only, without changing the default for subsequent calls

### Requirement: Raw session never mutated by redaction
Redaction SHALL never modify the original raw session file; it SHALL only affect derived artifacts (filtered transcript, handoff, on-demand expand output).

#### Scenario: Raw file is untouched after redaction runs
- **WHEN** redaction is applied while producing a filtered transcript
- **THEN** the original raw session file SHALL remain byte-for-byte unchanged

### Requirement: No secret data leaves the configured local endpoint
The system SHALL NOT transmit session data (redacted or not) to any network destination other than the explicitly configured local OpenAI-compatible endpoint, and SHALL NOT collect or transmit telemetry.

#### Scenario: No telemetry call is made
- **WHEN** any command in this feature runs, including handoff generation
- **THEN** no network request SHALL be made to any host other than the configured `qwen.base_url` (and, for MCP/CLI operations that don't call the distiller, no network request SHALL be made at all)
