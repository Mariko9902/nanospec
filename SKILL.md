---
name: nanospec
description: Write, review, validate, revise, or execute NanoSpec development specifications. Use for Cycles, Stories, execution state, acceptance evidence, and the nanospec CLI.
---

# NanoSpec

Choose the mode for the current request. Read only the rules and templates it needs. The caller's instructions and the target project's rules govern project-specific work.

## Boundaries

- A Story is the unit of specification, execution, and acceptance. A Cycle organizes Stories without hiding shared requirements outside them.
- `depends_on`, Acceptance Status, in-progress Evidence, and Outcome hold the facts. The Dependency Graph displays current facts from `cycle.md`; it is a generated view.
- Human acceptance is the default. `verified` requires explicit authority, scope, coverage review, and independent verification.
- Acceptance records and artifacts stay in each Story's `acceptance/` directory. Use `evidence.md` during execution, `outcome.md` after acceptance, and `evidence/` for artifacts.
- The CLI checks structure, assembles context, and records state. It does not select work, implement it, or grant acceptance authority.

## Write

1. Read the [concept model](./concept-model.md), [decomposition rules](./decomposition-rules.md), and [rule selection rules](./rule-selection-rules.md).
2. Set `spec_language` from the explicit request, then the existing Cycle, then the default `en-US`. Record it in the Cycle. A request such as "write the spec in Japanese" is enough. Check the source requirements and actual project before choosing boundaries or dependencies.
3. Read the templates needed for this work: [Cycle](./cycle-spec-template.md), [Story](./story-spec-template.md), [implementation brief](./implementation-brief-template.md).
4. Run `nanospec validate --sync-views <cycle-dir>` to check the draft and generate views.
5. For the first Cycle draft, use a separate review context to read the original requirements, project facts, and complete draft. Follow the [review scope](./splitting-checklist.md#review-scope) for later changes.

## Review and validate

- Run `nanospec validate <cycle-dir>` for read-only structural checks. Use `--sync-views` only when views need updating.
- Use `nanospec show overview <cycle-dir>` for titles and scenarios in dependency order, then read the relevant full specifications.
- For semantic review, read the [checklist](./splitting-checklist.md), source requirements, project facts, affected specifications, and current state. A passing CLI check does not prove a correct specification or an accepted result.
- Check the selected document language and the meaning of translations. Fixed English headings, field names, enums, and identifiers stay unchanged.
- For delivery review, follow the [execution rules](./execution-rules.md). Keep implementation checks, independent verification, and final acceptance separate.

## Revise

Read the [change rules](./spec-change-rules.md) when scope, decisions, dependencies, or requirements change. Read the [feedback rules](./feedback-loop-rules.md) only when the problem may belong to NanoSpec itself.

Keep accepted Stories and Outcomes frozen. Update the requirements baseline and affected unfinished work. Run `nanospec revise --reason <text> <affected-story>...` once for all affected Stories in a Cycle. It validates the changes, updates views, advances the revision, carries forward current evidence, and makes only the selected evidence stale.

Repartition work only when every affected Story is entirely unstarted and every Acceptance condition maps to the new boundaries without changing meaning. For a regression after acceptance, create a new Story with `regression_of` pointing to the frozen result.

## Execute

1. Run `nanospec prepare <story-dir-or-story.md>...` for the caller's explicit selection. Graph position and `nanospec ready` do not select work or imply a batch.
2. Use each execution package as the starting input. It includes the embedded execution rules, specification, state, dependencies, language, and startup conditions. Check the listed environment, evidence, availability, and human acceptance requirements against the actual project.
3. A Story marked `waiting_for_selected_dependency` waits until its selected prerequisite is `ready` or `accepted` and its result is available in the current environment.
4. Read the [execution rules](./execution-rules.md) while implementing, observing, recording, reviewing, or handling a blocker.
5. Keep the prepared `spec_revision`. Before each execution write, pass it as `--spec-revision`; a mismatch requires fresh preparation and impact assessment. Save a checkpoint when work starts or resumes. Use `--issue` for a blocked checkpoint. Use `--complete --next none` only when implementation has ended. Rejected work starts repair with `--status rework`; blocked work preserves a Resume Status and must restore that stage after resolving the cause.
6. Record each observation with `nanospec evidence`, including the command, actual result, artifact path, code version, environment, spec revision, and time.
7. Once all observations are complete and implementation is marked complete, run `nanospec verify` with the actual verifier and `self` or `independent` mode. A `rework` result records rejection and waits for an actual repair assignment.
8. Run `nanospec accept` only after explicit acceptance or after every condition of an existing `verified` authorization is met. A passed check does not supply that authority.
9. Use `nanospec reject` for a rejected `ready` result, then continue from the recorded Lane and next action.

Use `nanospec prepare --json` when a caller needs the structured execution package. Preparation is read-only.
