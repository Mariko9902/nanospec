# Rule Selection

Select rules that change the current Story's business behavior, interaction, or acceptance result. Keep stack, directory, build, and implementation requirements in the target project's rules.

## Inputs

Rule sources must be explicit:

```yaml
rule_sources:
  - <rule-pool-path>
```

Use `rule_sources: []` when no rule pool is supplied. Do not guess a personal directory or rule-library path. If a source cannot be read, report that it did not take part in selection.

A rule can use these fields:

```yaml
id: miniapp-list-infinite-scroll
target: story
scope:
  platform: miniapp
applies_when:
  page_type: list
strength: required
rule: Append the next page at the end of the list. Stop requesting pages when none remain.
```

- `id` is a stable reference.
- `target` is `story` or `execution`.
- `scope` and `applies_when` define when the rule applies.
- `strength` is `required` or `default`.
- `rule` states the concrete requirement.

An applicable `required` rule cannot be silently dropped. An explicit Story decision can replace a `default` rule. Split mixed business and implementation rules at the source.

## Selection

After the Story boundary and a first Description are clear:

1. Identify the platform, domain, roles, entries, actions, states, and data from the Story and confirmed project facts.
2. Use the source's index, directories, tags, or search to narrow the pool. Read only plausible matches for `scope` and `applies_when`.
3. Exclude `target: execution` rules and rules that do not change this Story's behavior or acceptance result.
4. Check the remaining rules against the original requirements, actual project, confirmed decisions, and one another.
5. Put the selected IDs and concrete local bodies in `Definition/Applicable Rules`. Reflect observable outcomes in Acceptance without adding a check mechanically for every rule.

Select for each Story. Judge candidates from all sources together; source order does not establish priority.

## Conflicts

- A `required` rule conflicts with a confirmed requirement: record the conflict in Open Questions and ask for a decision.
- An explicit decision replaces a `default` rule: omit that rule from Applicable Rules. Record a useful exception in Decisions when traceability matters.
- A rule disagrees with the project: investigate an implementation defect, an outdated rule, or a migration in progress. Do not pick a convenient side.
- Applicability or competing rules remain uncertain: keep an assumption or question instead of presenting a candidate as fact.

## Local snapshots

```md
### Applicable Rules

Source: `docs/mobile-list-rules.md`, version `<checked-version>`.

- `miniapp-list-infinite-scroll`: Append the next page when the list reaches its end. When no pages remain, stop requesting and show the end-of-list state.
```

The ID provides traceability. The local body supplies the meaning needed to execute the Story: current objects, conditions, strength, outcomes, and exceptions. An ID, link, or instruction to follow another document is not enough. Record only sources and versions actually checked, once per source. Use the Story's language for the body and keep IDs and source references unchanged.

Keep one primary expression for each rule. External rules belong in Applicable Rules; decisions made by the current requirements belong in Business Logic; observable conditions belong in Acceptance. Description keeps the process readable. Considerations holds unsettled counterexamples worth checking.

A changed rule pool does not rewrite a Story. Find the affected unfinished Stories, update their snapshots, validate, and run `nanospec revise` when meaning changes. Correcting a source path or checked version alone does not increment `spec_rev`. Accepted records stay frozen; a newer source cannot silently replace them.
