# MCP Tools

Start the stdio server:

```bash
cc-session serve-mcp --config ~/.session-context/config.json
```

Tools:

- `list_sessions`
- `inspect_session`
- `filter_session`
- `create_handoff`
- `get_handoff`
- `search_session`
- `expand_evidence`
- `compare_context_size`
- `verify_workspace`

The MCP server calls `internal/broker` directly. It does not shell out to the CLI. Multiple agent clients may launch independent stdio server processes against the same evidence store.
