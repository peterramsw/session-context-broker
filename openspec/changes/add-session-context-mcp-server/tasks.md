## 1. Shared Core

- [x] 1.1 Extract provider resolution, inspect, filter, stats, and handoff policy into shared internal broker/core packages
- [x] 1.2 Keep CLI command files limited to argument parsing and output formatting
- [x] 1.3 Add tests proving CLI and core outputs match for the same sessions

## 2. MCP Server

- [x] 2.1 Evaluate `modelcontextprotocol/go-sdk`, falling back to `mark3labs/mcp-go` only if necessary
- [x] 2.2 Implement `cc-session serve-mcp`
- [x] 2.3 Expose `list_sessions`, `inspect_session`, `filter_session`, `create_handoff`, `get_handoff`, `search_session`, `expand_evidence`, `compare_context_size`, and `verify_workspace`
- [x] 2.4 Add stdio server smoke test and tool schema tests

## 3. Workspace Verification

- [x] 3.1 Implement read-only git inspection only
- [x] 3.2 Enforce `allowed_workspace_roots`
- [x] 3.3 Add tests that no workspace files or git refs mutate

## 4. Concurrency

- [x] 4.1 Run three concurrent `serve-mcp` processes against the same store
- [x] 4.2 Prove one slow operation does not block unrelated reads
