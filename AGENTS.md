# Session Context Broker Agent Guide

全新獨立專案，範疇/技術棧待補（使用者稍後會補充細節）。這份文件是本專案的最高規則文件，之後補充專案細節時請直接編輯本檔，不要另開一份取代它。

## Start Here

1. Run `git status --short --branch` before any new task (if the repo has been git-initialized).
2. List `openspec/changes/` and read only the active change artifacts relevant to the task.
3. Read this file in full before starting non-trivial work; keep it short — put detailed architecture/file maps in separate docs (e.g. `ARCHITECTURE.md` / `REPO_MAP.md`) once the repo has real structure to describe.

## Repo Entry Points

TBD — populate once the source layout is decided (main module(s), key entry files, build/run instructions).

## OpenSpec Workflow

This repo uses [OpenSpec](https://github.com/Fission-AI/OpenSpec) (`openspec/`) for spec-driven change management. Conventions below mirror the workflow used in the sibling project `D:\repo\typingnote\AGENTS.md` — read that file if you need more detail on any rule.

- Use OpenSpec for non-trivial product, architecture, or behavior changes. Trivial fixes/typos don't need a change proposal.
- Treat `openspec/specs/**/spec.md` as the current accepted behavior baseline. `openspec/changes/archive/` is history only — never treat it as current truth.
- **Discuss first, don't draft.** When the user brings a requirement or problem, inventory it and align on direction before creating any OpenSpec artifacts. Do not immediately call `/opsx:new` or `/opsx:propose`. Exception: if the user explicitly says "直接寫 change" / "skip discussion", skip straight to drafting.
- When starting a new change, use the OpenSpec OPSX workflow/tooling (`/opsx:new`, or `/opsx:propose` if the user wants all artifacts drafted at once), not manual change-directory creation.
- Before implementing a behavior change, read the relevant files under:
  - `openspec/changes/<change-id>/proposal.md`
  - `openspec/changes/<change-id>/design.md` when present
  - `openspec/changes/<change-id>/specs/**/spec.md`
  - `openspec/changes/<change-id>/tasks.md`
- Keep `tasks.md` in sync while implementing.
- Do NOT write tool outputs or generated context directly into `openspec/specs/`.

### Close-out (收尾)

When the user says `收尾`, follow the close-out workflow **for each change separately** (one sync + one archive + one commit per change — do NOT bulk-archive multiple changes in a single commit; that loses per-change traceability):

1. Finish and update the change artifacts (`tasks.md`, and any proposal/design/spec delta files that changed during implementation).
2. Sync accepted delta specs into `openspec/specs/` using the OpenSpec sync tooling (`openspec-sync-specs` skill / OpenSpec CLI) — not by manually copying files.
3. Archive the completed change using the OpenSpec archive tooling (`openspec-archive-change` skill / OpenSpec CLI) — not by manually moving directories.
4. Commit the sync + archive together with message `chore(openspec): archive <change-name>`. One change per commit.
5. After all in-batch archives are committed, run a cross-cutting doc review pass once: check whether this file (and any `ARCHITECTURE.md` / `REPO_MAP.md` that exist by then) still match reality, and commit any updates separately from the per-change archive commits.

## Guardrails

- Keep this file short. Once the repo grows real structure, split detailed architecture/file maps into separate docs rather than expanding this file indefinitely.
- New features or behavior changes must go through `openspec/changes/<change-id>/`.
- CodeGraph (`codegraph_*`) is the preferred structural code-understanding tool. If `.codegraph/` does not exist for this repo, run `codegraph init -i` before structural code work, then verify with `codegraph_status`.
