## 1. Shared Core

- [x] 1.1 Extract provider resolution, inspect, filter, stats, and handoff policy into shared internal broker/core packages
- [x] 1.2 Keep CLI command files limited to argument parsing and output formatting
- [x] 1.3 Add tests proving CLI and core outputs match for the same sessions
- [x] 1.4 Inject a `HeaderScanner` into the broker store so `list_sessions` scans JSONL transcript headers when session-meta is sparse (fixed in `broker.New`)

## 2. MCP Server (official SDK)

- [ ] 2.1 Adopt the official `modelcontextprotocol/go-sdk` as the MCP transport/runtime and remove the hand-rolled JSON-RPC loop in `serve_mcp.go` (the interim newline-delimited framing fix is a stopgap until this lands)
- [ ] 2.2 Reimplement `cc-session serve-mcp` on the SDK, keeping `internal/broker` as the tool backend so CLI and MCP still share core logic
- [ ] 2.3 Expose `list_sessions`, `inspect_session`, `filter_session`, `create_handoff`, `get_handoff`, `search_session`, `expand_evidence`, `compare_context_size`, and `verify_workspace` with **typed** input schemas (replace `additionalProperties: true`)
- [ ] 2.4 Add an end-to-end test that drives the server as a real MCP client (initialize → tools/list → tools/call) over newline-delimited stdio and asserts parsed JSON-RPC responses — not substring matches on self-framed output
- [ ] 2.5 Structure the transport layer so a future Streamable HTTP transport can be added without changing tool handlers

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
