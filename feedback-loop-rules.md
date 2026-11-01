# Improving NanoSpec

A problem in one delivery does not always belong to the spec method. Use this file to decide where a correction belongs and when the method itself needs to change.

## 1. What is worth recording

Record feedback when a Story keeps drifting during execution, an otherwise complete spec leaves a necessary boundary unclear, acceptance cannot be judged from the evidence, an inseparable result keeps being split apart, dependencies cannot describe real readiness, state cannot restore the current work, or templates add repeated content without helping a decision.

Also record a validator that accepts a mechanical contradiction or blocks a valid spec, and any route that calls work complete without the required observations or authority.

An ordinary implementation defect, missing field, or one agent's misunderstanding may be an instance problem rather than a NanoSpec problem.

## 2. Find the level

1. **execution**: current progress, a blocker, or a temporary action.
2. **instance**: a mistake in the Cycle, Story, brief, or state.
3. **nanospec-rule**: a concept or rule likely to cause the same mistake elsewhere.
4. **nanospec-template**: a field, example, or instruction that invites the mistake.
5. **nanospec-tool**: a stable mechanical contradiction missed by the CLI, or a valid form it rejects.

Handle the first two through the [change rules](./spec-change-rules.md). The last three may require a change to this repository.

## 3. Keep the record small

Use the current Cycle's Practice Feedback section:

```md
## Practice Feedback

- Observed Problem: <What actually happened.>
  - Spec Impact: instance | nanospec-rule | nanospec-template | nanospec-tool
  - Proposed Fix: <The correction and where it belongs.>
  - Status: open | evaluating | fixed | rejected
```

Acceptance observations belong in the Story's evidence record and, once accepted, Outcome. A requirements mismatch before acceptance belongs in the unfinished Story. Feedback keeps the method question that still needs a decision.

Keep only issues still needing attention or a few just resolved. A separate permanent feedback log is unnecessary.

## 4. Diagnose the cause

Before changing the method, establish what happened, whether another reasonable executor would face the same ambiguity, and whether the cause is missing requirements, an outdated brief, inadequate state, or a wrong concept.

Check the practical cost of leaving it: a changed goal, wrong acceptance, lost recovery state, or wasted context. Then ask whether the proposed fix would prevent that cost without adding competing documents or more prose to every Story.

Separate these cases:

- A Story lacks requirements: fix the Story, not the brief.
- Repository findings are outdated: update the brief or investigate the current code.
- The implementation plan changed: the specification may be unchanged.
- A new result can ship independently: add a Story.
- Completion was declared too early: fix the evidence, acceptance state, or completion rules.

A machine check needs a stable rule with a low false-positive rate. Subjective business judgment remains a review task.

## 5. Compare useful facts

Outcome records `first_pass`, `rework_count`, and `spec_change_count`. Compare Stories only when their risks, spec depth, and project environments are reasonably similar, and read the actual Outcomes and feedback alongside the counts.

Repeated rework can point to mixed delivery boundaries, weak acceptance observations, or missing execution context. Repeated specification changes after work starts can point to insufficient project investigation, clarification, or rule selection. Repeated regressions against the same Outcome can expose a missed condition, check, or real entry point.

One count does not prove a method good or bad, and does not rate an executor. NanoSpec does not collect elapsed-time metrics. Change global guidance when comparable cases show a pattern or when one case exposes a serious structural error.

## 6. When the method should change

A NanoSpec change is warranted when the problem repeats across instances, a rule directly encourages wrong behavior, a rare mistake can cause a serious scope or acceptance error, or a specific correction removes ambiguity or wasted context across projects.

It is also warranted when a stable mechanical check can catch a problem without frequent false alarms.

Project-specific constraints, missing context in one request, an agent ignoring an explicit rule, subjective judgments with no general test, and fields that add form without changing decisions should stay out of the global method.

## 7. Make the smallest correction

- Concept confusion: update the concept model and related rules or templates.
- Bad delivery boundaries: update decomposition guidance and the review checklist.
- Missing execution or recovery input: update execution rules or the state template.
- A misleading template: fix the template and check the validator.
- A stable mechanical inconsistency: fix the validator and add a regression check.
- A single instance mistake: fix the instance.

Search for old terms and fields after a change. Keep the concepts, examples, templates, references, and validator consistent.

## 8. Close the feedback

Mark feedback `fixed` only when the cause is known, the correction is made, affected references and examples agree, and the relevant validation has passed.

Check wording changes for meaning and links. Exercise an example after a template or structural change. For a CLI change, run the relevant regression checks and validate the installed build.

Keep stable requirements in the spec and temporary implementation planning with the executor. If the issue is not a method problem, mark it `rejected` with the reason. If the practical effect still needs observation, keep it `evaluating`.

Use the installed `nanospec validate` for specification instances. Read the CLI source and build it when maintaining the tool itself. A passing structural check shows only that the checked structure holds; it does not establish that a method change has improved real delivery.
