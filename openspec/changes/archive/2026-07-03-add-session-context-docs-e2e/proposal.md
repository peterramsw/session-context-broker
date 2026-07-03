## Why

After the core implementation is split into focused changes, the project still needs user-facing setup docs, config docs, repeatable end-to-end validation, and a final evidence-backed report before a release-quality handoff.

## What Changes

- Complete config loading documentation and examples.
- Write architecture, provider, schema, Local LLM, MCP, Skills, security, and upstream-sync docs.
- Update README with fork attribution and preserved upstream behavior.
- Add opt-in live Local LLM integration test and full pipeline e2e validation.
- Produce the final evidence-backed report.

## Scope Boundary

This change should not introduce major runtime behavior beyond tests/docs/config polish.
