# Close Session

Create a compact handoff before ending substantial work:

1. Run `cc-session handoff <session-id> --llm auto`.
2. If Local LLM is unavailable or skipped, keep the filtered artifact and evidence index.
3. Record tests, commits, blockers, and next actions only when supported by evidence IDs or current workspace verification.
4. Do not copy raw transcript bodies into rule files or specs.
