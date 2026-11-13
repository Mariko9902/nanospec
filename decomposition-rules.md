# Writing a Spec

Turn source requirements and project facts into a Cycle, complete Stories, optional implementation briefs, and execution state. The [concept model](./concept-model.md) defines what those objects mean.

## Spec depth

Start lightweight. A local result with a simple entry, few rules, clear dependencies, and no substantial risk usually needs Orientation, Description, and Acceptance. It still needs a complete account of how the result happens.

Use full depth when the work involves multiple roles, permissions, sensitive operations or data, tenant boundaries or data scopes, money, billing, stock, quotas, idempotency, concurrency, audit, state transitions, cancellation, retries, recovery, asynchronous events, migration, compatibility, cross-system effects, new public contracts, multiple entry points or consumers jointly determining one result, critical ambiguity in the source requirements, or a conflict between the requirements and the actual project.

Both depths use the same format, Story identity, dependencies, Acceptance Flow, stable IDs, and acceptance rules. There is no `spec_depth` field. Full depth means expanding the parts that explain the actual risk. It does not mean filling every optional section. Lightweight work cannot omit a necessary condition just to stay short.

## Core principles

### Inspect the project before splitting the work

Read project rules and product context. Check actual code, data, entries, configuration, and behavior. Identify deliverable results and the rules needed for each. Then choose boundaries and blocking dependencies, build the Cycle order, and assign Story IDs. Write a complete local definition for each Story. Add a brief only when there are repository findings worth handing over.

Use `story-NN-<slug>`. The number is a stable handle, not an encoding of priority or dependency. For a new Cycle, assign it after boundaries and the initial dependency order are clear. Once implementation or verification facts exist, keep the number. A later order change does not renumber work. A meaning-preserving repartition of entirely unstarted work follows the [change rules](./spec-change-rules.md#repartitioning-unstarted-stories).

### A Story is one deliverable result

Ask whether the result can be handed to one agent, implemented, demonstrated, and accepted on its own. Its local requirements must be complete and the result must be meaningful.

If two candidates need to explain the same inseparable invariant together, the boundary may have cut through one result. If a candidate combines several results that could be accepted separately, it may be too large.

A file change, layer, or internal step is not a Story by itself. Keep necessary implementation steps inside the executor's temporary plan. Keep useful, nonbinding repository findings in the brief.

Story size is not measured by session length, line count, or anticipated code volume. Remove irrelevant requirements and repeated project rules first. Then look for independent results. Keep necessary rules together when they determine the same result, even when the rule set is dense.

### Follow the result from trigger to use

Identify who initiates the work, where it begins, which system areas participate, how data and states change, who consumes the result, and where it can be observed.

Every necessary part must be:

- Implemented and defined by this Story.
- A confirmed existing feature, with its use explained here.
- A prerequisite result, with an explicit dependency and connection.

A necessary part that is independently useful can be a prerequisite Story. Keep the rest of an inseparable result together.

Plan the main useful path first. Later variants and improvements can be separate work when independently deliverable. Required permissions, integrity, security, and compliance belong in the result from the start.

Infrastructure work can also be a Story: project setup, migration, a permission mechanism, audit logging, a shared state machine, a public upload API, an amount calculation, or system-wide regression. It still needs a result that can be accepted, not just a completed internal step.

### A Cycle organizes

Keep the Story list, dependencies, possible parallel work, priority, coordination, and Cycle-level risks in the Cycle. Put each Story's complete requirements in that Story. The next agent should not need to reconstruct all the other work to understand the selected result.

### Dependencies describe blocking facts

`depends_on` means the current result needs an actual result from a prerequisite. Numbering, preferred order, and resource allocation do not establish that need.

A prerequisite must be verified and available. It need not be human-accepted unless an explicit requirement says so. See [dependency readiness](./execution-rules.md#dependency-readiness).

For example, order placement depends on customer, product, and delivery-address results. Customer and product work can run in parallel. Order placement can begin once its own prerequisites are ready, even while an unrelated saved-products feature remains unfinished. Put a useful priority such as starting customer records first in Planning Context.

A review feature needs customer and product records because reviews refer to them. If only completed purchasers may review, it also needs the relevant order result. The business fact establishes the dependency.

### Put new results in the formal plan

When execution finds more work, decide what changed:

1. A temporary implementation step: handle it inside the current execution.
2. A repository fact or suggestion: update the optional brief when useful.
3. Scope, a rule, or acceptance for unfinished work: revise that Story.
4. Another independent result: create a Story.
5. Work outside the planning window: create a Cycle.
6. Update state to match the actual situation.

## Workflow

### Step 1: Establish the context

Set `spec_language` from the explicit request, then the existing Cycle, then `en-US`. Record the value in `cycle.md`. A clear language name or regional language tag is enough. Ask when the choice is unclear. Do not infer it from the source requirements or template language.

Read the project's `AGENTS.md`, `CLAUDE.md`, README, architecture decisions, product context, configuration, models, APIs, and relevant code. Check live behavior where needed.

Establish what the project is, what exists, where the relevant entries and data are, which rules apply, what the requested change would do, which contracts belong to other modules, and what cannot yet be confirmed.

Even a project with little code needs a product boundary, initial results, technical choices, directory and data conventions, a way to run, and usable acceptance entries.

Keep stable engineering rules in the project's own rule system. Do not create a NanoSpec-specific shared rules layer for them. Put the current Story's behavior in the Story and genuine uncertainty in the ledger.

If the work changes the stack, framework, module ownership, authentication, audit, error handling, external service, shared API, or data rules, check whether a project-level contract must change before splitting Stories.

Do not repeat obvious repository exclusions, treat a template as a project fact, invent a build or verification command, assert unconfirmed behavior, or guess a rule-pool location.

### Step 2: Choose boundaries

For each candidate, identify the independently useful result, its actual entry, observable acceptance, necessary participants, and how it is supplied by current work, existing behavior, or prerequisites.

Check whether adjacent candidates connect through available results or still need to explain one inseparable rule together. Look for mixed results, repeated project constraints, conflicting expectations, and requirements that are easy to lose separately. Keep the necessary constraints when they belong to the same result.

Use recognizable candidate names and temporary references first. Do not assign formal numbers until boundaries and order are clear. Add a dependency only when the result cannot stand without that prerequisite.

### Step 3: Select applicable rules

The caller supplies sources explicitly:

```yaml
rule_sources:
  - <personal-rule-pool-path>
  - <project-rule-pool-path>
```

Use `rule_sources: []` when none are supplied.

For each Story, identify its platform, domain, roles, actions, states, and data; read plausible matches; and follow [rule selection](./rule-selection-rules.md). Keep the selected IDs, concrete local bodies, and useful provenance. Keep engineering requirements with the project. Record unresolved conflicts in Open Questions. Reflect observable outcomes in Acceptance.

Do not copy the pool or make the executor reread it to understand the Story.

### Step 4: Build the Cycle

1. Build the candidate dependency DAG from actual blocking facts.
2. Choose priority among eligible Stories using the main path, unlocked value, early risk discovery, feedback, workload, and available resources.
3. For a new Cycle, assign formal IDs in the initial dependency order and replace temporary references. Keep existing IDs in an established Cycle, except for an eligible unstarted repartition.
4. Create every Story directory; the CLI discovers Cycle membership from those directories.
5. Put non-obvious priorities and coordination constraints in Planning Context.
6. Keep the Dependency Graph heading and generate its contents after drafting the Cycle execution facts.
7. Explain a useful priority change without renumbering Stories.

The Cycle plan lists work, possible parallelism, and coordination. The graph shows the smallest set of edges with the same reachability. Neither repeats Story definitions or acceptance.

### Step 5: Write each Story

The required parts are Orientation with Title and Scenario, Definition with Description, and Acceptance with a flow and stable observable conditions.

Add Business Logic, Out of Scope, and Decision Ledger only when they contain useful independent content. Keep any unresolved decision that would change delivery in Open Questions.

Within Definition, add Concepts before Description when needed, then the appropriate Business Logic, State Model, Flow Diagram, Constraints, UI Draft, Integration Notes, Applicable Rules, or Considerations.

The [Story template](./story-spec-template.md) is the complete authoring interface. Place settled local rules in Business Logic, complex relationships in the appropriate dedicated form, plausible wrong extrapolations in Constraints, existing-system connections in Integration Notes, external business rules in Applicable Rules, and unsettled counterexamples in Considerations. Keep each rule in one primary place.

Write a concrete Scenario in a warm, natural, patient voice before adding markers. Show an actual participant or trigger, the system's contribution, and how the result lets work continue. A shared mechanism can use one representative case, but the case must not hide its wider scope.

Mark complete business objects with `『』` and clear business actions with `「」`. Generic words such as record, reason, or result are not emphasis labels. Parallel objects may share markers when each remains clear. Keep conditions before actions and direct results with their actions. The full rules are in [Orientation](./story-spec-template.md#2-orientation).

A downstream Story explains its actual connection to an available upstream result. The upstream Story does not add empty placeholders for future work.

### Step 6: Decide whether a brief helps

After the Story's requirements are clear, look at the repository analysis already done. Keep only findings worth handing to the next agent: a reusable implementation, a subtle behavior, an easy-to-miss caller, a generated-code source, a public type, or a data relationship.

Do not let a candidate implementation define the Story's requirements. Do not create a brief merely for search hints or an unsupported idea.

Use `implementation_brief: none` when there is no worthwhile finding. Otherwise, create `implementation-brief.md` in the Story directory and reference that filename.

Repository Findings is required. Candidate Directions is optional, supported by findings, and last. State the checked sources, relevance, adoption conditions, and known limits. The brief adds information beyond the Story, PRD, Cycle, state, project rules, and ordinary code entries.

A contract that constrains delivery belongs in the Story or project rules. A code source that explains a finding belongs in the brief. Business risks belong in the Story; an approach's risk belongs with that candidate direction. The executor chooses the actual plan after rechecking the code.

### Step 7: Create state

Keep execution facts in `cycle.md`, with exactly one Acceptance Status block for every Story. Start at `rev-1`, usually with Cycle status `active` and `pending` acceptance. Record actual initial facts or `none` for the situation and issues. Active Lanes is `none` until execution starts; a Lane's Next then holds its recovery action.

Run `nanospec validate --sync-views <cycle-dir>` to check the draft and generate the dependency graph and status labels. Keep operation rules out of state. Execution starts through `nanospec prepare`, which includes the execution rules and selected context.

### Step 8: Review independently

After the first Cycle draft passes structural checks, a separate review context reads the source requirements, relevant project facts, and the complete draft. For later edits, use the [review scope](./splitting-checklist.md#review-scope).

Fix findings in their proper source. Put decisions still needed from a person in the affected Story's Open Questions. Check the corrections and their consequences, then validate. Keep trusted conclusions for unchanged work and expand the scope if the effect reaches further.

## Before handing it over

Confirm that every Story is one meaningful, independently acceptable result and accounts for every necessary participant and connection. Keep its requirements complete locally. Keep only real blocking facts in dependencies.

Acceptance starts with a Mermaid route through preparation, the actual entry, main and important non-success paths, observations, evidence, and the decision. Nodes mark their Acceptance IDs and cover every condition. A provider without a separate page needs a delivered call example or other executable entry, an observable execution result, and, when data is saved, a way to verify the persisted result.

The flow organizes rules already defined. Checks can provide repeatable operations when a real command and environment exist. They support only their actual observations and do not grant final acceptance. The executor performs available steps, records what cannot be done, and keeps unresolved observations pending. Human acceptance remains the default.

Confirmed repository facts go in Confirmed Facts, temporary interpretations in Assumptions, and decisions that need a person in Open Questions.

A finished draft should let a person select work by a stable ID and let another agent understand the result, investigate the current code, implement it, observe it, and collect evidence. The plan should show dependencies and useful priority without making the agent reconstruct every other Story.
