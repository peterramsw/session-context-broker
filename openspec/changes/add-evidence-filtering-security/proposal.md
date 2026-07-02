## Why

The first session-context slice proves multi-provider parsing and filtered-first handoff generation, but the production-grade safety layer is still incomplete. The tool still needs a real evidence store, stronger deterministic filtering guarantees, complete secret redaction, and semantic handoff validation before MCP/Skill workflows can safely rely on evidence IDs instead of re-reading large transcripts.

## What Changes

- Add a file-based evidence store under `storage_root` with manifest, normalized events, filtered output, evidence index, and handoff artifacts.
- Extend deterministic filtering so risk/decision signals are retained and compressed tool calls expose structured evidence summaries.
- Harden redaction and apply it at filtered-output and evidence-expansion boundaries.
- Complete handoff semantic validation: evidence-ref existence, unevidenced-claim demotion, deployment/rollback conflict detection, branch/commit mismatch warning.
- Add mock/local tests for failed repair, hallucinated evidence, chunked distillation merge, redaction, traversal protection, and concurrent writes.

## Scope Boundary

This change deliberately does not add MCP tools or Skills. It prepares the evidence/safety core those surfaces will call.
