# Resume Session

Use `cc-session handoff <session-id>` first. Prefer the filtered artifact as the default source; only expand evidence IDs when a claim matters for the next action.

Do not trust claims about tests, deployment, rollback, branch, or commit unless the handoff provides evidence refs and those refs are expanded or the workspace is re-verified.

When resuming:

1. Read the handoff or filtered transcript.
2. Expand evidence for high-risk claims.
3. Run `cc-session verify-workspace <path>` when branch, commit, or dirty state affects the answer.
4. Continue from verified facts and list claims that still require re-verification.
