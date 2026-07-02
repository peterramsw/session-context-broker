## Why

The provider/filtered-first CLI slice proves the core value, but agents need a programmatic surface to reuse that value without re-reading whole transcripts. MCP should expose that core as thin tools while preserving the local, lightweight architecture.

## What Changes

- Extract shared broker/core functions used by both CLI and MCP.
- Add `cc-session serve-mcp` stdio server with session, handoff, evidence, context-size, and workspace-verification tools.
- Add read-only workspace verification constrained by `allowed_workspace_roots`.
- Add multi-process concurrency tests for three independent client processes sharing the same evidence store.

## Scope Boundary

This change does not author the Skill workflows. Skills are a separate wrapper over this MCP tool surface.
