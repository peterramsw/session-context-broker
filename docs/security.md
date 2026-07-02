# Security

Security boundaries:

- Raw session files are read-only inputs and are never mutated.
- Filtered artifacts, Local LLM inputs, and evidence expansion responses are redacted by default.
- Unredacted evidence expansion requires an explicit flag/config path and is off by default.
- Evidence expansion re-reads source bytes by evidence ID, applies size limits, reports truncation, and rejects paths outside allowed roots.
- Evidence writes use temp-file-then-rename atomic writes and a short per-session write lock.
- Local LLM calls happen outside evidence write locks.
- `verify_workspace` only runs read-only git inspection and refuses paths outside `allowed_workspace_roots`.
