# Normalized Event Schema

Normalized events are Go `session.SessionEvent` values persisted as `normalized.jsonl`.

Important fields:

- `event_id`: stable ID derived from provider, session ID, sequence, source path, source line, byte offset, and event type.
- `session_id`, `provider`, `timestamp`, `sequence`
- `role`: `user` or `assistant` when applicable.
- `event_type`: examples include `message`, `tool_call`, `tool_result`, `reasoning`, `session_meta`, `unknown`.
- `content`: normalized text or compact JSON.
- `tool`: call ID, name, namespace, arguments, result/stdout/stderr, status, exit code.
- `source`: path, line range, byte range, and content hash.
- `metadata`: provider-specific extra fields.

Evidence IDs are derived from normalized event identity, not from filtered text, so IDs stay stable when summaries or redaction change.
