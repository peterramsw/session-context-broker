## ADDED Requirements

### Requirement: New subcommands added without breaking existing ones
The system SHALL add `list --provider {claude_code|codex|all}`, `inspect`, `filter`, `handoff` (with `--force` and `--provider`), `search`, `expand <evidence-id>`, `verify-workspace --workspace <path>`, and `serve-mcp` to the existing `cc-session` binary, while the existing `list`, `read`, `context`, `stats`, `audit`, `expand`, `inject` commands SHALL continue to behave exactly as before for their existing argument forms.

#### Scenario: Existing commands are unaffected
- **WHEN** the existing upstream CLI test suite (280 tests as of this change) is run after adding the new subcommands
- **THEN** every previously-passing test SHALL still pass with no changes to expected output

#### Scenario: New provider filter works
- **WHEN** `cc-session list --provider codex` is run
- **THEN** only Codex-provider sessions SHALL be listed

### Requirement: CLI and MCP share one core, no duplicated implementation
CLI subcommands SHALL call the same internal packages that back the MCP tools; command-specific code SHALL be limited to argument parsing and output formatting.

#### Scenario: A bug fix in core logic fixes both surfaces
- **WHEN** a defect in evidence expansion or handoff validation is fixed in the shared internal package
- **THEN** both `cc-session expand`/`cc-session handoff` and the corresponding MCP tools SHALL reflect the fix without separate patches

### Requirement: New commands work without a Local LLM endpoint
`list`, `inspect`, `filter`, `search`, `expand`, `stats`, `verify-workspace`, and `serve-mcp` (for tools other than `create_handoff`/`get_handoff`) SHALL function correctly with no Local LLM endpoint configured or reachable.

#### Scenario: Filter and search work offline
- **WHEN** `local_llm.base_url` is unset
- **THEN** `cc-session filter <session>` and `cc-session search <session> "<query>"` SHALL still succeed

#### Scenario: Handoff auto mode falls back to filtered output
- **WHEN** `cc-session handoff --llm auto <session>` is run without a configured Local LLM endpoint
- **THEN** the command SHALL write the deterministic `filtered.md` artifact and SHALL NOT fail solely because Local LLM is unavailable

#### Scenario: Handoff Local LLM mode is explicit
- **WHEN** `cc-session handoff --llm always <session>` is run without a configured Local LLM endpoint
- **THEN** the command SHALL return an actionable Local LLM configuration error after writing the filtered artifact
