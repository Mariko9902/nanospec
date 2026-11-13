# Cycle Spec Template

A Cycle groups Stories for one planning window and keeps their current execution and acceptance facts in `cycle.md`. Each Story keeps its own requirements. Dependencies come from `depends_on`; the CLI builds the graph and its status labels from those dependencies and the current records.

````md
---
id: cycle-01-<slug>
spec_language: en-US
window:
  - <planning-window>
spec_rev: rev-1
status: active
last_updated_at: <ISO-8601>
---

## Planning Context

- <Why this work belongs in the same window.>
- <Priority, coordination, or timing that matters beyond the dependency edges.>
- <Checks of existing environments, permissions, or prerequisite results; none if unnecessary.>

## Dependency Graph

- none

## Risks

- <A risk for this Cycle; none if there are no known risks.>

## Current Situation

- none

## Active Lanes

- none

## Acceptance Status

- Story: story-01-<slug-a>
  - Acceptance: pending: <Observed or unconfirmed Acceptance IDs, context, and evidence path.>
- Story: story-02-<slug-b>
  - Acceptance: pending: <Observed or unconfirmed Acceptance IDs, context, and evidence path.>

## Open Issues

- none

## Practice Feedback

- none
````

## Planning and dependencies

Place `cycle.md` in a numbered directory such as `spec/cycle-01-orders/`, beside the Story directories. Write `spec_language` from the explicit request or use `en-US` for a new Cycle. Keep fixed headings, field names, enums, identifiers, commands, and filenames unchanged. Prose follows the Cycle's language.

The Story directories define Cycle membership. The CLI includes every Story in the graph, including isolated nodes, and omits dependency edges implied by another path. Node order is deterministic and puts prerequisites before dependents. It does not select work, assign priority, or require a batch to finish before independent work can continue.

Describe a non-obvious priority or coordination constraint in Planning Context. It does not create a dependency. Business rules, setup work still needed, and acceptance conditions belong to the relevant Stories.

After drafting the Cycle and Stories, run:

```sh
nanospec validate --sync-views <cycle-dir>
```

The command rebuilds the graph and Cycle summary. Each node starts with exactly one status label, such as `[rework] S01 Save an order`. The seven Story stages are:

| Status | Meaning |
| --- | --- |
| pending | Not started or normally waiting for dependencies or selection. |
| active | First implementation or its self-checks are underway. |
| ready | Executor checks passed; ready for independent review and final acceptance. |
| rejected | Review or acceptance refused the result; waiting for a rework assignment. |
| rework | The executor has explicitly started the assigned repair, including its checks. |
| blocked | A necessary condition is missing; the cause and recovery action are recorded. |
| accepted | The authorized acceptance occurred and Outcome is frozen. |

Normal work follows `pending -> active -> ready -> accepted`. Returned work follows `ready -> rejected -> rework -> ready`. AI and human refusals use the same rejected stage; who made the decision remains in the review and acceptance facts. Rejection does not start rework. Preparation and task dispatch do not start it either; the executor records a rework checkpoint when actually taking the assignment.

Colors distinguish the same seven stages, and the labels remain visible without color. Details do not add a second status tag. Graph edits cannot change execution or acceptance facts.

## Execution and acceptance

Use `nanospec prepare` before starting, resuming, verifying, or accepting a Story. Retain its `spec_revision` as the execution baseline. Pass that exact value through `--spec-revision` to `checkpoint`, `evidence`, `verify`, `accept`, and `reject`. A different current revision refuses the write. Rerun preparation, assess the changed Story and prerequisite results, and adjust implementation and evidence before continuing.

A revision change does not automatically invalidate every Story. `nanospec revise` advances the Cycle revision once, updates in-progress evidence to that revision, and makes only the explicitly affected evidence stale. Started affected work records a recovery action to reassess the new specification. Active work becomes blocked; rejected work remains rejected until rework actually starts. Existing blockers keep their cause and recovery action. Passed checks affected by the revision no longer permit a ready recovery. Resuming requires fresh preparation and a checkpoint. Determine the affected set from actual behavior and contracts, including related unfinished work; the dependency graph alone cannot establish semantic impact. Unaffected evidence may continue after that assessment. Progress and ordinary evidence updates do not increase `spec_rev`.

Current Situation and Open Issues keep only the facts needed to resume. Each Lane keeps its executor and next action. Replace obsolete entries rather than accumulating a journal or duplicate evidence. Evidence and Outcome remain under each Story's `acceptance/` directory.

Every Story has exactly one Acceptance Status block. Its state is `pending`, `ready`, `accepted`, or `rejected`. Explanations identify observation and review coverage, Acceptance IDs, context, evidence locations, and required human confirmations. Record separately obtained prerequisite confirmations in the current Story's explanation before using those results. Complete accepted results belong in Outcome.

Cycle `status` is derived: `done` when every Story is accepted, `blocked` when every unfinished Story is blocked, and `active` otherwise. It summarizes the planning window, not running processes. Use the CLI to change Story facts; a Lane has this form:

```md
- Lane: lane-01
  - Story: story-01-<slug-a>
  - Owner: <executor-id>
  - Status: active
  - Next: <The next action.>
  - Implementation Complete: false
  - Spec Changes: 0
```

`Owner` uses the same stable identifier format as `implemented_by`, with no spaces. A pending Lane records an assignment before implementation starts and does not accrue specification-change counts. Owner can change before evidence exists. After `acceptance/evidence.md` exists, a checkpoint cannot directly change it. A checkpoint with `--complete --next none` records that implementation has ended. It keeps the active or rework stage until verification passes; it does not accept the result. Acceptance removes the finished Lane.

Verification returning `rework` sets the same Lane to rejected and records the assignment still needed. Human rejection does the same. The next executor starts with `checkpoint --status rework`; only then can it record new observations. Acceptance Status keeps the latest acceptance facts separately from the current execution stage.

Active, rework, and ready checkpoints require resolved Open Questions and prerequisites currently ready or accepted. A blocked prerequisite does not qualify even if its earlier checks passed. Normal dependency waiting stays pending until an explicitly selected task records a blocker.

A blocked checkpoint requires an issue and recovery action. A caller-requested hold of selected work can use the same record until the caller releases it. It saves the prior stage in `Resume Status`, for example rework or ready. Repeating the checkpoint preserves that target. When the condition is resolved, checkpoint back to that stage; a pending task may explicitly start active, and a rejected task may explicitly start its assigned rework. A ready recovery also checks that its implementation and passed evidence remain current. Do not use blocking or recovery to skip stages. Clear the resolved issue when restoring progress.

Only blocked Lanes contain `Resume Status`. All Lanes contain `Implementation Complete: true | false`. The complete flag is recorded during active or rework and retained when checks pass into ready. Blocking preserves the prior completion fact; rejection or a semantic revision affecting the implementation clears it. The flag alone does not establish ready or authorize acceptance. All execution writes require the prepared revision, including blocking and recovery.

Keep method feedback only while it needs attention or affects later work:

```md
- Observed Problem: <What happened.>
  - Spec Impact: instance | nanospec-rule | nanospec-template | nanospec-tool
  - Proposed Fix: <Where to fix it and what to change.>
  - Status: open | evaluating | fixed | rejected
```

## Repartition Records

For a meaning-preserving repartition of entirely unstarted Stories, place this optional section between Dependency Graph and Risks. Record new, unused IDs, the current revision, and the complete Acceptance mapping. Accepted Stories and Outcomes remain frozen. Changed acceptance meaning follows the ordinary change rules.

```md
## Repartition Records

- Repartition: repartition-01
  - Spec Revision: rev-2
  - Before: [story-01-old, story-02-old]
  - After: [story-03-new, story-04-new]
  - Acceptance Map: story-01-old/A1-A2 -> story-03-new/A1-A2; story-02-old/A1 -> story-04-new/A1
  - Reason: <Why the boundaries are easier to execute or accept independently.>
```
