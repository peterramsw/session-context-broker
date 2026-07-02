## Context

This phase keeps the original cc-session-reader design principle: reduce token cost by filtering noisy transcript data before it enters an agent context. The implementation remains CLI-first and local-file based, with Local LLM handoff generation as an optional derived-artifact layer.

## Decisions

**1. Provider adapters are lightweight and file-format specific.**
Codex and Antigravity parsing live in separate adapter packages. Codex targets local Codex rollout JSONL sessions. Antigravity targets Google's standalone app brain store under `~/.gemini/antigravity/brain`, not Antigravity IDE storage.

**2. Filtered-first handoff policy.**
`handoff` always writes a redacted `filtered.md` artifact before any Local LLM decision. In `--llm auto`, the decision uses redacted filtered character count, not raw session size. The default threshold is intentionally simple (`8000`) and configurable.

**3. Local LLM is optional and deterministic by default.**
The OpenAI-compatible client is disabled unless configured. Requests use `temperature: 0` by default and accept explicit `max_context`, `max_output_tokens`, `top_p`, and `top_k` settings. Users without Local LLM can still use provider listing, inspect, filter, stats, and filtered-only handoff.

**4. Handoff is derived, filtered transcript remains truth.**
The generated `handoff.json`/`handoff.md` is a resume accelerator. `handoff.md` includes the required derived-artifact disclosure. The filtered transcript remains the source to inspect when claims matter.

**5. Follow-up work is intentionally split.**
Evidence IDs/store, MCP server, Skills, installer onboarding, stronger redaction, and full e2e docs are not in this phase. Splitting them keeps the token-saving CLI usable and prevents the first working slice from becoming over-engineered.

## Risks / Trade-offs

- Antigravity parsing is based on the standalone app brain store verified on this machine; Antigravity IDE storage is explicitly out of scope for this phase.
- Local LLM output can need repair and may still be wrong; it is treated as derived and optional.
- Some provider support is still wired through CLI branches; the shared broker core is deferred to the MCP follow-up change.
