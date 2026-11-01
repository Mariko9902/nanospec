# Implementation Brief Template

`implementation-brief.md` passes on repository findings worth keeping and possible approaches supported by those findings. Create it only when it saves substantial rediscovery or prevents a likely mistake.

Use the Cycle's `spec_language` for prose. Keep fixed headings, paths, and code identifiers unchanged.

The Story defines the result and its constraints. The brief supplies repository facts and suggestions. If they conflict, requirements stay in the Story and repository facts must be checked against the live code.

At least one `Repository Finding` is required. Otherwise, set `implementation_brief: none` in the Story and omit this file. Each Story can have one brief.

```md
# Implementation Brief

<Optional short context needed to understand the findings.>

## Repository Findings

- `<path-or-path#symbol>`: <A confirmed fact, its source, and why it matters to this Story.>

## Candidate Directions

- <An approach supported by the findings, why it may work, what needs checking, and when it may fail.>
```

## Writing rules

For `Repository Findings`:

- Include facts that take real investigation to rediscover or are easy to miss or misread.
- Identify the source and its relevance. Mention useful existing implementations first.
- Where a fact depends on a version or environment, record the checked revision, content digest, or environment and time.
- Useful subjects include callers, generated-code sources, public types, data relationships, behavior, and confirmed code risks.
- Group longer findings by the actual business object, system area, module, or call relationship.
- Public contracts and architecture decisions that constrain the result belong in the Story or project rules. They may also appear here as a source for a finding.

For `Candidate Directions`:

- Keep only approaches supported by findings. Put this section last, and omit it when empty.
- State the evidence, adoption conditions, and known reasons an approach may fail.
- The executing agent checks the current repository before deciding what to do.

Keep other content where it belongs:

- Requirements, business rules, acceptance conditions, and scope: `story.md`.
- Temporary implementation steps: the executor's current plan.
- Execution progress and recovery: `execution-state.md`.
- PRD, Cycle, project rules, and other source material: their original files.
- Ordinary entry-point discovery: the executing agent's repository investigation.

The brief adds information beyond those sources. Do not repeat them. A search hint becomes a finding only after it has been checked and shown to be worth handing over.

The executor can inspect additional callers and files, or choose another implementation. A change to the Story's behavior, acceptance, public contracts, or scope follows the [change rules](./spec-change-rules.md).
