# Handoff Schema

`handoff.json` uses schema version `session-context-handoff/v1`.

Top-level sections:

- `session`: provider, session ID, source path, workspace, model, raw and filtered chars.
- `objective`
- `confirmed_decisions`
- `rejected_or_superseded`
- `implementation_state`
- `verification`: passed, failed, not run, warnings.
- `deployment`: completed flag, environment, evidence refs, rollback claims.
- `known_blockers`
- `unresolved_questions`
- `next_actions`
- `user_corrections`
- `claims_requiring_reverification`
- `workflow_improvement_candidates`
- `validation`: warnings and conflicts.

Claims that matter for tests, deployment, rollback, branch, commit, blockers, and corrections should carry resolvable evidence refs. Unknown evidence refs are stripped with warnings.
