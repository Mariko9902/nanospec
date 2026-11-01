# Execution Rules

These rules apply when an agent receives a Story. The caller's instructions, project `AGENTS.md`, and other project rules still govern the worktree, tools, accounts, environment, and project-specific requirements.

## 1. Execution unit

Execute a Story as one defined result. Keep the temporary implementation plan in the current session. Record progress, blockers, recovery actions, and acceptance facts in `execution-state.md`; keep the stable definition in the Story.

Before starting, check that:

- The Story can be understood on its own.
- Prerequisite evidence applies and the required results are available here.
- Open Questions have been resolved.
- The project is ready for implementation.
- The work can stay within the defined behavior and scope.

Stop when a necessary condition is missing or the work requires a change to scope, business rules, dependencies, or acceptance. Follow the [change rules](./spec-change-rules.md).

Treat uncertainty according to its effect:

- Different answers change behavior, boundaries, rules, Acceptance, or Out of Scope: ask for a requirements decision before implementing.
- The unknown is a file location, current implementation, API capability, dependency behavior, or implementation approach: investigate the project first.
- Different answers do not change the delivery: continue within the confirmed project and Story constraints without inventing an Open Question.

When the remaining unknowns can be resolved by inspecting the project, finish clarification and begin implementation.

### Dependency readiness

Before starting or resuming, check every `depends_on` prerequisite:

- Its acceptance state is `ready` or `accepted`, and its state explanation or Outcome contains evidence still applicable to the result needed here.
- The required code, service, data, or API is available in the actual execution environment. An implementation on another branch or evidence from another environment is not enough by itself.
- If the project or Story requires prior human acceptance, confirm the human-accepted Outcome or obtain an explicit confirmation for the result being used. Record any separately obtained human confirmation in the current Story's state explanation before proceeding. Acceptance under a verification policy does not supply a required human decision.

A prerequisite may remain `ready` when these checks pass. Its progress stays unchecked and no Outcome is created early. Execute only the work the caller selected.

`regression_of` supplies frozen background. It does not pass on the old acceptance conclusion or replace new evidence. A warning about missing artifacts or context describes a limit in the old record; it does not mean the missing material exists or that current requirements can be reduced.

Record the prerequisite version or entry being used in the current Story's state explanation. Shared environment information can refer to an existing record. If a prerequisite returns to `pending`, is rejected, or its required result changes or becomes invalid, pause affected work and check the consequences. Update the blocker and next action. Resume once the conditions hold again.

`nanospec verify --result pass` and `nanospec accept` recheck Open Questions and recorded prerequisite states. If the check fails, they do not advance review or acceptance and do not freeze an Outcome. Environment and evidence applicability still require a real check.

## 2. Execution context

Run `nanospec prepare` for every explicitly selected Story before starting or resuming. Each package combines these execution rules with the current specification, state, dependencies, language, and startup conditions. Judge each selected Story separately. A selected prerequisite that is not ready still blocks its dependent Story.

`Spec Language`, or `spec_language` in JSON, gives the Cycle's document language. The default is American English, `en-US`. Use it for specification changes, situation and next-action values, issues, decisions, and observation explanations. The language of this rule file or the conversation does not override it. Keep fixed headings, fields, enums, IDs, paths, commands, and raw output unchanged. CLI summaries and record labels use the protocol's English text.

The setting governs document language. Product UI language, currency, time zone, and local business rules follow the Story's explicit requirements. A language change never rewrites an accepted Story, brief, or Outcome.

Read as needed:

1. Project rules, public contracts, and supplied product context.
2. The current `story.md`.
3. Its optional `implementation-brief.md`, for additional repository findings.
4. The current `execution-state.md`.
5. Code, configuration, data, and running behavior discovered while checking the Story.

Read `cycle.md` when dependencies, ordering, or coordination need confirmation. Do not load every Story by default or rely on another Story's prose as an unstated requirement.

Applicable Rules already contains selected rule bodies. Execution does not require rescanning the whole pool. Source links support traceability; their latest content cannot silently replace a snapshot. If a snapshot conflicts with current mandatory project rules or explicit instructions, handle the conflict through the change rules.

### Shared edits and state updates

When work is arranged in parallel, check for incompatible edits to the same API, data definition, migration, or file. A lack of business dependency does not prevent a file conflict. Sequence the conflicting edits through the execution arrangement instead of inventing a `depends_on` edge.

Keep state current:

- Use a checkpoint when starting, resuming, pausing, blocking, or ending implementation. It reads and validates the current state before updating the selected Story's Lane, situation, issues, and timestamp.
- Writes to one Cycle run in sequence across reading, validation, and saving. If the wait exceeds 10 seconds, the command fails. Reread the state before trying again; do not overwrite another executor's facts by hand.
- `Owner` is the current executor's stable identifier and follows the same format as `implemented_by`. It can change before evidence exists. Once `acceptance/evidence.md` exists, a checkpoint rejects a direct change. Continue under the original identifier without rewriting implementation responsibility.
- `Current Situation` holds current facts. Each Lane holds its Story, Owner, Status, and Next action. One Lane's status is not the entire Cycle's status.
- Replace obsolete situation entries. Keep accepted results in Outcome rather than accumulating an execution journal, commit log, or duplicate evidence.
- `Acceptance Status` records the stage, covered or unconfirmed Acceptance IDs, observation context, and evidence location. Outcome holds the full accepted result.
- Acceptance writes the current `spec_rev` and exact Story digest. A changed accepted Story fails validation. Fix a later regression with a new Story.
- If `story.md` changes between acceptance preparation and saving, acceptance refuses to freeze or restores the original evidence and state. Recheck the specification before trying again.
- Use rejection to record a final refusal of a ready result. It returns the evidence for rework, increments the rework count, and pauses the Lane. A new observation returns acceptance to `pending`.
- Keep only unresolved issues and method feedback that still needs attention. Remove a blocker when it is resolved.
- After editing formal execution meaning, run `nanospec revise` once for every affected Story in that Cycle. It advances the revision and the started Stories' change counts. All in-progress records follow the new revision; only selected evidence becomes stale. The caller still determines the affected set.
- When the revision changes, check the definition, implementation, and evidence. Old observations keep their original context. Reobserve affected conditions.
- Synchronize views after changing order, dependencies, or acceptance state. A view does not create a new decision or evidence.

State copies in different worktrees do not synchronize themselves. Reconcile them through the project's normal collaboration process before using a combined state.

### Acceptance files

Keep records in each Story's `acceptance/` directory: `evidence.md` during implementation, `outcome.md` after acceptance, and raw screenshots, responses, query results, or reports under `evidence/`. Save material there from the first observation. Acceptance freezes the record without moving the artifacts.

Use paths relative to `acceptance/`, such as `evidence/requests.json`. Shared artifacts can reference the owning Story. Keep one acceptance conclusion instead of copying it into a separate evidence tree at the project root.

## 3. Using an implementation brief

The brief passes on findings checked during planning and candidate approaches supported by them.

- Recheck the paths, symbols, interfaces, behavior, and conclusions in Repository Findings against the live code.
- Then consider whether the suggested directions still fit their stated conditions.
- Use another approach when it produces the same required result more appropriately.
- Inspect additional related files when the implementation needs it.

If a brief disagrees with the repository, the repository determines the code facts. Correct a location or suggestion in the brief. If the conflict shows that the Story's requirement is wrong or needs to change, follow the change rules.

## 4. Work loop

### Inspect

Read the Story, optional brief, state, and project rules. Inspect actual entries, callers, data, APIs, and existing behavior. Identify the gap between the required result and the system as it stands.

### Reconcile

Separate facts, assumptions, and decisions still needed from a person. Distinguish missing requirements from repository questions that need more investigation. Check whether implementation would change the Story's scope, public behavior, or exclusions. Resolve a blocker or specification change before proceeding.

### Act

Resolve Open Questions and write clarification decisions into the formal requirements baseline. Treat the current Acceptance conditions as the implementation baseline. Make a small temporary plan from the current code, then implement the complete result needed for those conditions. Follow the project's module boundaries, contracts, and change rules.

### Observe

- Record the Cycle revision, code revision or artifact digest, environment entry, and observation time.
- Prepare the accounts, business data, and prerequisites in Acceptance Flow. Follow its actual entry points and operations.
- Perform every step available within the current environment and authority. If a step cannot run, record the missing permission, input, or external condition and its effect. Keep the result `pending` instead of skipping or assuming success.
- Check the real business flow, including the page, API, persisted data, and related states.
- Observe the main result and the boundaries or non-success conditions defined in the Story.
- Keep enough page, response, export, or persisted-result evidence to support each Acceptance ID.

For an Executable Check, confirm the command, environment, and Expected value still match the current version before running it. A check supports only the IDs and observations it actually covers. An unreferenced check, an environment mismatch, or partial coverage cannot establish the rest of Acceptance.

A repeatable check can directly provide business observation when it operates through the actual page, API, or delivered call entry and checks the business result. Confirm its inputs, expectations, coverage, and environment. Do not repeat an already observed operation solely to produce another record format.

Syntax, type, and formatting checks establish their own limited conclusions. Tests against mocks or substitute services do not establish uncovered behavior in the real system. A log saying "passed" cannot stand in for all Acceptance items. Match verification depth to Story risk, caller instructions, and project rules.

### Review

Review the actual result and evidence. The implementer performs their own checks first. Where the project, Story, or authorization requires independent verification, a verifier who did not produce the implementation checks that scope. A separate review of the written specification does not replace delivery verification.

- Check that preparation, main paths, important non-success cases, and evidence collection were covered.
- Review every Acceptance item and Out of Scope boundary.
- Connect each observed result to its execution entry and supporting evidence.
- Check whether later specification, code, or environment changes affect an observation. Reverify the affected conditions before continuing to rely on a pass.
- Identify anything unobserved, undecidable, or different from the specification.
- Record implementer, actual verifier, verification mode, time, and covered IDs. The implementer cannot claim independence by opening another session.

### Record

Use the lifecycle commands for the facts they own:

- `checkpoint`: current situation, next action, and blocker.
- `evidence`: each current observation in `acceptance/evidence.md`. Reobserving an ID replaces its current fact instead of adding a journal entry.
- `verify`: verifier, mode, and passed or rework result after all observations. A pass reaches `ready`.
- `accept`: an explicit human decision, or a valid declared verification authorization with every condition met. It freezes Outcome and updates acceptance and progress.
- `reject`: a human refusal of a ready result. Do not manually rewrite acceptance or evidence to imitate it.

Record whether the first complete review passed, how many rework rounds returned to implementation, and the Lane's specification-change count. These improve the method; they do not track time or rate the executor.

## 5. Scope decisions

Internal restructuring, extra repository investigation, another implementation approach, and necessary checks can stay within the Story when the public result is unchanged.

Stop and handle a specification change when the work would add or remove public behavior, change Definition, Acceptance, or Out of Scope, change an actual blocking dependency, require a missing human decision, or change another independent delivery.

Implementation difficulty, existing behavior, limited checks, and completed code are not reasons to lower Acceptance. A new product decision can change an unfinished Story, but the baseline must be updated before implementation continues.

## 6. Completion

A Story is complete only when all Acceptance items have been observed, the evidence supports the decision, the required acceptance has occurred, and `acceptance/outcome.md` exists.

With the default policy, verification can move the Story to `ready`. Human acceptance requires an explicit decision before running `nanospec accept --method human`.

A `verified` policy requires traceable authorization, scope, coverage review, and independent verification of all Acceptance items by a verifier other than the implementer. If any condition is unclear, invalid, or unmet, keep the Story `pending` or `ready`. Do not accept it. A failed review is not a reason to change policy or reduce the conditions for success.
