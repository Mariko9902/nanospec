# Execution State Template

`execution-state.md` holds current facts. Obtain the execution rules and selected context with `nanospec prepare` before starting or resuming a Story. Use the CLI to change state.

Write situation, issue, next-action, and feedback values in the Cycle's `spec_language`. Keep headings, field names, enums, and fixed CLI summaries unchanged.

```md
---
cycle: <cycle-id>
spec_rev: <rev-N>
status: active | blocked | paused | done
last_updated_at: <ISO-8601>
---

## Progress Tree

- [ ] story-01-<slug-a>
- [ ] story-02-<slug-b>

## Current Situation

- <The latest fact needed to resume work; none before work starts.>

## Active Lanes

- none

## Acceptance Status

- Story: story-01-<slug-a>
  - Acceptance: pending: <Observed or unconfirmed Acceptance IDs, context, and evidence path.>
- Story: story-02-<slug-b>
  - Acceptance: pending: <Observed or unconfirmed Acceptance IDs, context, and evidence path.>

## Open Issues

- <An issue still affecting progress or acceptance; none if clear.>

## Practice Feedback

- none
```

## State rules

- Start `spec_rev` at `rev-1`. Increment it for changes to formal execution meaning, not for progress or evidence updates.
- Cycle `status` is `active`, `paused`, `blocked`, or `done`. It can be `done` only when every Story is accepted.
- The CLI generates `Progress Tree` from Wave order and Acceptance Status. A checked Story must be accepted and have a complete Outcome.
- `Current Situation` and `Open Issues` keep the facts needed to resume work. A Lane's `Next` holds its recovery action. Replace obsolete facts instead of accumulating a journal, commit history, or full evidence.
- Every Story has exactly one acceptance block. Its state is `pending`, `ready`, `accepted`, or a `rejected` state recorded by the rejection command. `accepted` is final.
- Acceptance explanations identify the current observation and review coverage, context, Acceptance IDs, evidence location, and any required human confirmation. Record separately obtained confirmations of prerequisite results in the current Story's state explanation before using those results. Complete accepted results belong in Outcome.
- Use a checkpoint to create or update a Story's Lane, situation, blocker, and timestamp. Do not assemble Lanes by hand.
- `active`, `paused`, and `done` checkpoints require clear Open Questions and prerequisites at `ready` or `accepted`. A `blocked` checkpoint can record an unmet condition.
- After a semantic edit passes validation, run `nanospec revise` once for all affected Stories in the same Cycle. It advances `spec_rev` and the started Stories' spec-change counts.

A Lane has this form:

```md
- Lane: lane-01
  - Story: story-01-<slug>
  - Owner: <executor-id>
  - Status: active | paused | blocked | done
  - Next: <The next action; none if implementation has ended.>
  - Spec Changes: <Semantic changes since implementation started, initially 0.>
```

Each Lane points to one Story. Checkpoints update that Story's state. Acceptance removes its finished Lane. A done Lane does not mean the Story has been accepted.

`Owner` is the stable identifier of the executor responsible for continuing the Story. It follows the same format as `implemented_by`, with no spaces. It can change before evidence exists. Once `acceptance/evidence.md` exists, a direct change is rejected; the original Owner can continue recording evidence.

Keep method feedback only while it still needs attention or affects later work:

```md
- Observed Problem: <What happened.>
  - Spec Impact: instance | nanospec-rule | nanospec-template | nanospec-tool
  - Proposed Fix: <Where to fix it and what to change.>
  - Status: open | evaluating | fixed | rejected
```

After drafting the state, run `nanospec validate --sync-views <cycle-dir>`.
