# Design: Session Context MCP Server

## Decision

Build `cc-session serve-mcp` on the **official Go MCP SDK**
(`github.com/modelcontextprotocol/go-sdk` v1.6.1) rather than a hand-rolled
JSON-RPC loop.

The first pass hand-rolled the protocol and shipped two defects the self-framed
tests could not catch: LSP-style `Content-Length` framing instead of MCP's
newline-delimited stdio, and an `additionalProperties: true` catch-all schema
that made clients drop declared arguments (e.g. `list_sessions`' `limit`). The
SDK owns framing, initialize/capability negotiation, and schema inference, which
removes that whole class of bug and matches the original brief's requirement to
build on a maintained SDK.

## Architecture

```
runServeMCP(cfg)
  └─ broker.New(store, reader, cfg)          ← shared core (CLI + MCP)
     └─ mcpServer{svc, cfg}
        └─ server() *mcp.Server
             mcp.AddTool(srv, &Tool{...}, handler)   × 9
        └─ srv.Run(ctx, &mcp.StdioTransport{})
```

- **Tool backend is unchanged core.** Every handler is a thin adapter: unmarshal
  typed args → call an `internal/broker` method → wrap the result as text
  content. No provider/session/handoff logic lives in the MCP layer, so a bug fix
  in the core reaches both the CLI and MCP surfaces.
- **Typed input schemas by inference.** Each tool takes a typed Go arg struct
  (`listArgs`, `sessionArgs`, …). `mcp.AddTool` infers the JSON input schema from
  the struct — field names/types become properties, the `jsonschema` tag supplies
  descriptions. This is what makes clients forward all declared arguments.
- **Prefix resolution for stored artifacts.** `get_handoff` and `expand_evidence`
  read persisted artifacts, so they resolve a session-id prefix against the
  evidence-store directory (`resolveStoredSession`) instead of re-parsing the live
  session, and accept prefixes like every other tool.

## Transport

- **stdio now.** `mcp.StdioTransport` communicates over stdin/stdout with
  newline-delimited JSON — the transport Claude Code, Codex, and Antigravity use
  for local MCP servers.
- **Streamable HTTP later.** Because tool handlers are transport-agnostic, adding
  a remote transport (e.g. a GB10-hosted broker serving multiple machines) is a
  one-line transport swap in `runServeMCP`; handlers and the broker do not change.
  SSE is intentionally not used — it is deprecated in favour of Streamable HTTP.

## Testing

- **In-memory client↔server round-trip.** Tests connect a real `mcp.Client` to
  the server over `mcp.NewInMemoryTransports()` and exercise
  initialize → tools/list → tools/call — a genuine MCP handshake, not a
  self-framed harness. They assert the nine tools are advertised with typed
  schemas (including `list_sessions.limit`).
- **Concurrent clients.** Three client sessions run concurrently to confirm
  independent sessions coexist.
- **Real stdio** was additionally verified by driving the built binary with a
  persistent-stdin client (initialize → tools/list → inspect_session).

## Consequences

- Adds the first external dependency (`go-sdk` + transitive), so the module now
  has a `go.sum`.
- The SDK requires **Go 1.25**; `go.mod`, CI, and the release workflow are pinned
  to 1.25 accordingly.
