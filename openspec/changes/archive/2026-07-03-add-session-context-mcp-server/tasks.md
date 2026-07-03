## 1. Shared Core

- [x] 1.1 Extract provider resolution, inspect, filter, stats, and handoff policy into shared internal broker/core packages
- [x] 1.2 Keep CLI command files limited to argument parsing and output formatting
- [x] 1.3 Add tests proving CLI and core outputs match for the same sessions
- [x] 1.4 Inject a `HeaderScanner` into the broker store so `list_sessions` scans JSONL transcript headers when session-meta is sparse (fixed in `broker.New`)

## 2. MCP Server (official SDK)

- [x] 2.1 Adopt the official `modelcontextprotocol/go-sdk` (v1.6.1) as the MCP runtime and remove the hand-rolled JSON-RPC loop in `serve_mcp.go`; the interim framing fix is gone (SDK `StdioTransport` owns newline-delimited framing)
- [x] 2.2 Reimplement `cc-session serve-mcp` on the SDK, keeping `internal/broker` as the tool backend so CLI and MCP still share core logic (handlers only marshal args → broker calls → text result)
- [x] 2.3 Expose all nine tools with **typed** input schemas — the SDK infers each schema from a typed Go arg struct, so `additionalProperties: true` is gone and clients forward every declared argument
- [x] 2.4 Add an end-to-end test that drives the server as a real MCP client (initialize → tools/list → tools/call) via the SDK in-memory transport pair, asserting parsed results — not substring matches on self-framed output; also verified over real stdio
- [x] 2.5 Transport is swappable without touching handlers — tool handlers are transport-agnostic, so adding Streamable HTTP later is a one-line transport swap in `runServeMCP`

## 2b. Tool defects found in live MCP testing

- [x] 2b.1 `inspect_session` returns empty metadata — fixed: `buildClaudeMetadata` now derives user/assistant/tool/result and line counts from parsed events (verified: 13/22/19/19/91)
- [x] 2b.2 Resolve session-id prefixes consistently across every tool — fixed: `resolveStoredSession` maps a prefix to the persisted `<provider>/<full-id>` dir for `expand_evidence` and `get_handoff` (verified end-to-end with a prefix)
- [x] 2b.3 `list_sessions` ignored the `limit` argument — root cause was the untyped `additionalProperties:true` schema so clients dropped it; fixed by giving every tool a typed input schema (this also satisfies the typed-schema intent of 2.3 on the current server)

## 3. Workspace Verification

- [x] 3.1 Implement read-only git inspection only
- [x] 3.2 Enforce `allowed_workspace_roots`
- [x] 3.3 Add tests that no workspace files or git refs mutate

## 4. Concurrency

- [x] 4.1 Run three concurrent `serve-mcp` processes against the same store
- [x] 4.2 Prove one slow operation does not block unrelated reads
