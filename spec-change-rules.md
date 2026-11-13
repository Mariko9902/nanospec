# Spec Changes

Use these rules when execution uncovers a missing requirement, a changed project fact, a larger scope, or new work.

The files have different jobs:

- `cycle.md`: planning, dependency graph, specification revision, and current execution and acceptance facts.
- `story.md`: requirements, execution boundary, and acceptance.
- `implementation-brief.md`: optional repository findings and suggestions.
- `acceptance/outcome.md`: the accepted result.
- The PRD or designated product requirements baseline: the source product requirements used to write the Cycle.

## Ground rules

Temporary implementation work does not become a formal requirement by itself. An unfinished Story can change, with its consequences kept in sync. Accepted Stories and Outcomes stay frozen; later changes enter unfinished or new work.

Keep requirements in the Story and replaceable code findings in the brief. A changed file location is not a changed product requirement. A product change cannot live only in a brief.

## Clarifications and the product baseline

An answer in a conversation is not yet a shared requirement for implementation, review, and testing. Classify the answer first:

- **Product decision**: adds, changes, or confirms behavior, scope, roles, fields, states, boundaries, failure feedback, or acceptance expectations.
- **Story organization decision**: changes a dependency expression, acceptance expression, or local structure without changing product behavior or the expected result.
- **Repository fact or implementation judgment**: concerns code locations, internals, current behavior, dependency behavior, or a temporary approach.

For a product decision:

1. Remove the resolved question from Open Questions and record the answer in Decisions or Confirmed Facts.
2. Update every affected part of Definition, Acceptance, Out of Scope, and `depends_on`. Updating the ledger alone is not enough.
3. Write the stable product-level conclusion back to the PRD or designated baseline. Leave conversation transcripts and implementation detail out.
4. Check every related unfinished Story in the Cycle, including shared data, state, and rules as well as direct and indirect dependencies.
5. Update the affected Stories and plan, validate, and run `nanospec revise` once with the complete affected set.
6. Review the affected specifications under the [review checklist](./splitting-checklist.md), fix findings, and validate again.
7. Continue implementation only after the Story, product baseline, related Stories, and state agree.

A Story organization decision does not need a PRD edit, but it still belongs in the affected formal section. Increment `spec_rev` only if execution meaning changes. Resolving previously unspecified behavior, constraints, or acceptance expectations is such a change.

Repository facts do not rewrite product requirements. Correct the brief when useful, or use the finding in the current implementation decision.

If the answer only confirms a conclusion already explicit in the Story and PRD, remove the redundant question, keep any useful decision, and update state. Do not duplicate the requirement or increment the revision.

If a product-baseline update is required but its location, authority, or write access is unclear, stop and identify that missing input. Do not leave the conversation as the only source of truth.

New decisions about accepted behavior update the target product requirements and enter new Stories. Keep the original Story and Outcome unchanged. Their revision and content digest identify what was accepted.

A changed acceptance policy or authorization scope is a formal decision and increments `spec_rev`. Switching to `verified` requires prior explicit authority plus `acceptance_authority`, `acceptance_scope`, and `acceptance_review`. Switching to `human` removes those three fields. A failing review is not a reason to change the policy or lower acceptance conditions.

## Acceptance as the implementation baseline

Once Open Questions are resolved and the requirements baseline is synchronized, the current Acceptance conditions become the baseline for starting or resuming implementation.

An unfinished Story can still change through a new product decision. Stop implementation, update the Story, product baseline, related unfinished work, and the planning and execution facts in `cycle.md`, then continue.

These are not reasons to redefine success:

- The implementation is difficult.
- Existing code does something else.
- Current checks or evidence prove only a weaker result.
- Completed code would pass under a different standard.

Correct the implementation or evidence, explain why acceptance is unreachable, or request a product decision. A changed Acceptance definition needs a formal requirements decision or boundary change and a trace in the Decision Ledger.

## 1. Update state only

Use state updates for progress, blocking or recovery, external blockers, recovery actions, actual acceptance transitions, temporary implementation steps, and method feedback.

Read current state and update the selected entries, timestamp, and generated views through the relevant CLI commands. Do not create permanent work nodes for temporary actions or advance `spec_rev` for progress. Owner changes follow the evidence restrictions in the [Cycle execution state](./cycle-spec-template.md#execution-and-acceptance).

## 2. Update an implementation brief

For unfinished work, a brief-only change can correct a file, symbol, API, or table location; replace a finding that no longer matches the code; add a useful confirmed fact; or revise a candidate approach.

The Story's scope, rules, dependencies, and acceptance must remain unchanged. Keep the current `spec_rev`. Findings retain their checked sources; directions retain their evidence, adoption conditions, and failure conditions.

If a Story uses `implementation_brief: none`, creating the brief also changes that field to `implementation-brief.md`. This declares an auxiliary file and does not by itself change execution meaning. Removing an unhelpful brief reverses the field to `none` and follows the project's file-operation rules.

An accepted Story's brief stays frozen. Later code facts belong to the actual repository and later work.

## Repartitioning unstarted Stories

Work can be split, combined, or moved when the boundaries prove awkward, but only when every condition holds:

- All affected Stories are `pending`, with no Lane, implementation changes, evidence, or Outcome.
- The combined meaning of Definition, rules, Out of Scope, and Acceptance is fully preserved. Only ownership of requirements, boundaries, order, or direct dependencies changes.
- Every old Acceptance item maps to exactly one new item. No business result is added, dropped, weakened, or expanded.

Then:

1. Update the Stories, direct dependencies, plan, and state. Use unused IDs and never reuse retired numbers.
2. Add one Repartition Record to the Cycle with the revision, old and new IDs, complete Acceptance mapping, and reason.
3. Validate and run one revision listing every resulting Story. Synchronize views and review the affected scope. Do not rewrite unchanged product requirements as if new behavior had been added.

If any affected work has started or the mapping cannot prove unchanged meaning, use an ordinary Story change, new Story, or new Cycle instead.

## 3. Change an existing Story

Stop and edit an unfinished Story when execution meaning changes in:

- Description's goal, entries, defaults, or boundary.
- Business Logic, Constraints, or Applicable Rules.
- `depends_on`.
- Acceptance Flow or an Acceptance condition.
- An Executable Check's Expected result or references that change actual acceptance coverage.
- Out of Scope or a decision that affects acceptance.

Explain the difference and reason. Update all affected sections, then run:

```sh
nanospec revise --reason "<reason>" <affected-story>...
```

The command validates the specification, synchronizes views, advances the Cycle revision once, and updates the started Stories' change counts. Every in-progress record follows the new revision. Selected items become stale; removed or renamed Acceptance IDs are removed from their affected in-progress records. Check related unfinished work and review the affected Stories and relationships.

Wording, equivalent phrasing, and diagram layout changes do not advance `spec_rev` when meaning is unchanged. Changing a diagram direction is a layout edit; changing a preparation condition, required action, or pass condition is a semantic edit.

For a language change, update `spec_language` and express unfinished documents in that language. Check terminology, conditions, requirement strength, and acceptance meaning. Accepted Stories, briefs, and Outcomes keep their original text. Changes to product UI text, currency, time zone, or regional business rules are judged by their behavioral effect, not treated as mere translation.

A corrected check command, environment location, or artifact path does not advance the revision when Acceptance and Expected meaning remain unchanged. Recheck its coverage and rerun it where needed.

Apply the [review scope](./splitting-checklist.md#review-scope) to order, coordination, wording, and state changes. A priority change still advances `spec_rev`, but does not automatically require a separate full review or invalidate evidence.

When an external rule changes, find affected unfinished Stories by source, ID, applicability, or result. A source version change alone is not a reason to rewrite unrelated Stories. Correcting a checked source or shortening equivalent prose does not advance the revision. Changing conditions, strength, or exceptions does.

Internal code structure, another implementation approach, extra repository investigation, and necessary internal protection do not require a Story edit when the public result and exclusions stay unchanged.

## 4. Add a Story

Create a Story for a new result that can be delivered and accepted independently, belongs beyond a temporary implementation step, and cannot be expressed simply as a correction to the current unfinished result.

Examples include a feature, migration, setup result, stability improvement, regression fix, or a result previously excluded from scope.

1. Use the next unused Story number. Do not renumber existing work to insert it.
2. Create its Story file and an optional brief when justified.
3. Create its Story directory and give it an Acceptance Status block in `cycle.md`. Update the current facts and timestamp.
4. Validate and run one revision listing all affected Stories, including the new one.
5. Check related unfinished work, synchronize views, and independently review the new Story and affected relationships.
6. For the first Cycle or an unclear impact, review the complete draft. Fix findings, check their consequences, and validate again.

If the work belongs outside this planning window, create another Cycle.

For a regression after acceptance, set `regression_of: [cycle-id/story-id]` on the new Story. The target must have a Story and Outcome. Keep the old Story, brief, and Outcome frozen. Preparation checks their binding and includes them as read-only background. The reference does not inherit acceptance or establish a blocking dependency; use `depends_on` for a real prerequisite.

An external frozen human acceptance record may lack both artifact locations and evidence context for every item. It can be included with an explicit warning only if its other required content, digest binding, acceptance and verification facts, coverage, confirmations, observations, commands, and results all validate. Partial omissions, another missing field, a mismatched digest, a same-Cycle reference, or non-human acceptance must pass the complete checks. This exception does not establish that missing evidence ever existed or reduce current requirements. Preserve the original record.

## 5. Check related work

A semantic change requires checking every related unfinished Story in the Cycle, not just declared dependents. Include shared data, states, APIs, entries, roles, permissions, fields, boundaries, and rules.

Check that:

- Direct dependencies still describe the real prerequisites.
- Description and Integration Notes still match actual inputs, entries, APIs, and states.
- Business Logic, Constraints, and Applicable Rules agree.
- Acceptance can still stand on its own and Out of Scope still holds.
- Brief findings and directions still fit the code.
- Planning Context priorities, notes, and graph edges still describe the actual plan.
- Acceptance Status references every Story, and the graph displays every current state.

Downstream work uses a real upstream result. If that result changes, reassess the connection against the system as it now exists.

## 6. Accept a Story and freeze Outcome

Follow the [completion rules](./execution-rules.md#6-completion) and [Outcome format](./outcome-record-template.md):

1. Implement and observe the real business behavior. Record evidence for every Acceptance ID at the current revision.
2. Mark implementation complete with `checkpoint --complete --next none` and record verification with the actual verifier. A pass reaches `ready`, not completion.
3. Run acceptance only after the explicit human decision or every condition of the declared `verified` authorization has been established.
4. Acceptance freezes `evidence.md` as `outcome.md`, removes the Lane, and updates state and progress. The last accepted Story completes the Cycle.

Keep the accepted Story and brief frozen. Later work enters a new Story rather than rewriting the original plan.

## 7. Start another Cycle

Use another Cycle when the planning window or release changes, the current Cycle is complete, coordination or reporting needs a separate boundary, or another group of Stories has become the clearer plan.

Check the actual code, data, and behavior before choosing its scope.

## 8. Change NanoSpec

A repeated, structural, or high-risk problem in the method may need a change to the concept model, rules, templates, or validator. Record it in Practice Feedback and use the [feedback rules](./feedback-loop-rules.md). One incidental misunderstanding does not automatically justify another global rule.

## Change reference

| Finding | Update | `spec_rev` |
| --- | --- | --- |
| Progress, blocker, recovery, or next action | Execution fields in `cycle.md` | Unchanged |
| Repository location, fact, or suggestion | Brief | Unchanged |
| Equivalent wording, layout, or a redundant resolved question | The source document and relevant execution fields in `cycle.md` | Unchanged |
| A previously undecided requirement or a changed product decision | Story, product baseline, affected unfinished work, and `cycle.md` | Advance only affected Cycles with changed execution meaning |
| Scope, rules, dependencies, or acceptance | Story and `cycle.md` | Advance |
| Meaning-preserving repartition of unstarted work | Stories and the repartition record and execution fields in `cycle.md` | Advance |
| Execution priority or dependencies | Planning and execution fields in `cycle.md` | Advance |
| A new independent result | New Story and `cycle.md` | Advance |
| A regression after acceptance | New Story with `regression_of` and `cycle.md` | Advance |
| Accepted observations | Outcome and execution fields in `cycle.md` | No advance solely for recording a result |
| A method problem | Feedback and NanoSpec when warranted | Depends on execution meaning |

## Update order

Check the Story, product baseline, state, and actual code. Classify the change. Resolve any required human decision. Put its answer in the ledger and affected formal sections; write product decisions back to the product baseline. Check the full affected set when meaning changes, update the plan, validate, and run one revision for that set. Review the appropriate scope, fix findings, and validate again. Recheck the project before resuming implementation.
