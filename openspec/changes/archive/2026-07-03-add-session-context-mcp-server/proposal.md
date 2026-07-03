## Why

The provider/filtered-first CLI slice proves the core value, but agents need a programmatic surface to reuse that value without re-reading whole transcripts. MCP should expose that core as thin tools while preserving the local, lightweight architecture.

The first pass hand-rolled the JSON-RPC loop and shipped two protocol defects — LSP-style `Content-Length` framing instead of MCP's newline-delimited stdio, and `list_sessions` returning null because the broker store had no header scanner — that the self-framed tests did not catch. This confirms the original brief's requirement to build the MCP surface on a maintained SDK rather than hand-roll the protocol, especially since Claude Code, Codex, and Antigravity all consume this same server.

## What Changes

- Extract shared broker/core functions used by both CLI and MCP.
- Implement `cc-session serve-mcp` on the **official Go MCP SDK** (`github.com/modelcontextprotocol/go-sdk`) instead of a hand-rolled JSON-RPC loop, so stdio framing, initialize/capability negotiation, and typed tool schemas come from the SDK. **BREAKING** to `serve_mcp.go` internals; external tool names and behavior are unchanged.
- Expose the nine session/handoff/evidence/context-size/workspace tools with typed input schemas (replacing the current `additionalProperties: true` catch-all).
- Structure the transport so a future Streamable HTTP transport (remote / GB10-hosted broker) can be added without rewriting tool handlers.
- Add read-only workspace verification constrained by `allowed_workspace_roots`.
- Add multi-process concurrency tests for three independent client processes sharing the same evidence store.

## Scope Boundary

This change does not author the Skill workflows. Skills are a separate wrapper over this MCP tool surface. It also does not add the Streamable HTTP transport itself — only leaves room for it.
