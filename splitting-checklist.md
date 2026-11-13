# Semantic Review

Use the CLI for the mechanical checks first: directories, metadata, headings, references, Acceptance IDs, Outcomes, state, dependencies, graph status labels, and flow coverage. This checklist is for the meaning and risk those checks cannot settle.

## Review scope

| Change | Read and review |
| --- | --- |
| First Cycle, or no trusted review baseline | Source requirements, relevant project facts, and the full Cycle |
| New Story, changed scope, rules, contracts, dependencies, acceptance, policy, or authority | The diff, affected Stories in full, and related requirements and dependencies |
| Repartition of entirely unstarted work with unchanged meaning | Eligibility, the complete Acceptance mapping, dependencies, and new boundaries |
| Priority, assignment, or coordination only | The reason, actual dependencies, and edit conflicts |
| Wording, sources, progress, evidence, or views with unchanged meaning | The changed facts and local consistency |

Expand the scope when the effect is unclear or crosses shared data, states, APIs, roles, or rules. An independent review context reads the original sources, actual project, and specification. The author's summary cannot be its only evidence.

## Risk depth

A lightweight spec fits a local result with a simple entry, few rules, and clear dependencies. It still needs every applicable check below.

Use a full specification and review when the work involves multiple roles, permissions, sensitive operations or data, tenant boundaries or data scopes, financial amounts, billing, inventory, quotas, idempotency, concurrency, audit, state transitions, cancellation, retry, recovery, asynchronous events, migration, compatibility, cross-system effects, a new public contract, multiple entry points or consumers jointly determining one result, critical ambiguity in the source requirements, or a conflict between the requirements and the actual project.

Full depth means explaining the risks that actually matter. It does not mean filling every optional section. Lightweight work still needs a clear boundary, actual entry, observable result, and acceptance route.

## Delivery boundary

- Is there one result that can be understood, implemented, and accepted on its own?
- Has the main path or an inseparable business invariant been cut across Stories?
- Does the Story contain another independent result or treat an internal implementation step as a delivery?
- Have permissions, integrity, security, or compliance needed for a valid result been postponed as optional work?

## Human understanding

- Does the Cycle state `spec_language`, and does the prose read naturally in that language and region? Are the fixed English keywords, enums, identifiers, and exact quoted sources intact? Has translation kept facts, conditions, and requirement strength without silently changing product localization?
- Does the title identify the result quickly and match the generated views?
- Can someone new to the feature picture a concrete event and understand what this Story contributes? Is the scenario more than a sequence of clicks, processing, and display?
- Does it begin with an actual participant or trigger and show at least one useful relationship: a visible state change, retained unfinished work, a stable answer to uncertainty, or a result returned to the next task? Without a human role, is the downstream consumer clear?
- Are those relationships confirmed facts rather than invented convenience, emotion, or value? Has gentle wording preserved the full strength of permission, amount, state, failure, and prohibition rules?
- Do object and action markers make the relationships easier to read? Are conditions before actions, direct results kept with their actions, and semicolons used only between relatively independent actions?
- Are marked objects complete, identifiable business names rather than generic words such as record, reason, result, state, data, or content used for emphasis?
- Are parallel objects marked simply? When they share `『』`, is each one still clear, without conditions or action prose mixed into the object list?
- Without the markers, does the scenario still sound warm, natural, and patient, stay concise, and show the work the system takes care of? Do the markers support the explanation without taking priority over warmth and care?
- Does Orientation agree with Definition and Acceptance while staying a short human entry point?
- Are unfamiliar business terms explained in Concepts with plain words or a useful example, without circular definitions?
- Has Concepts stayed focused on business terms instead of becoming a glossary of ordinary technical mechanisms?
- Do explanations agree with project terms and the actual rules? Have examples kept necessary conditions and avoided new promises?
- Are the same business terms consistent across Stories, including accepted Stories, while accepted text remains frozen?

## Self-contained requirements

- Can the Story explain the goal, participants, entries, objects, process, defaults, and observation points without the planning conversation, another Story, or an external link?
- Does Description explain the problem being solved and the business value of the result, alongside the actual behavior?
- Is every necessary part supplied by this Story, a confirmed existing feature, or an explicit prerequisite?
- Do roles, fields, pages, APIs, and states match the actual project rather than assumptions from their names?
- Does the Cycle stay an organizer instead of a hidden shared requirements document?

## Rules and uncertainty

- Does each settled rule state its conditions, strength, outcome, and exceptions in one primary place?
- Do external rule snapshots keep only applicable IDs and local bodies? Do engineering rules remain in the project's rules?
- Are Considerations specific, grounded counterexamples rather than generic reminders or a second set of requirements?
- Are human decisions that could change the result in Open Questions? Are questions answerable through project investigation still being investigated?

## Acceptance and observation

- Does Acceptance cover every observable condition needed for the result and agree with Definition and Out of Scope?
- Can someone unfamiliar with the system follow the preparation, entry, actions, observations, evidence IDs, and failure route?
- Does the flow cover the main path and the defined important non-success cases without adding rules or prescribing internal implementation?
- Is success judged through the business result rather than files, code completion, a command's status, or the executor's account?
- Are Executable Checks real, available entries with stated Acceptance references, environment, and concrete expectations?
- Are implementation checks, independent verification, and acceptance recorded separately? Does a `verified` policy have actual authority, scope, coverage review, and a verifier who did not produce the implementation?

## Dependencies and coordination

- Does every `depends_on` edge mean the result cannot stand without the prerequisite's actual result?
- When a prerequisite requires human acceptance, is its human-accepted Outcome available, or is a separately obtained confirmation recorded in the current Story's state explanation?
- Does the downstream Story explain the connection to existing results without asking the upstream Story for an empty placeholder?
- Do the priorities in Planning Context remain distinct from dependency edges? Have downstream value, the main path, risk, feedback, workload, and available resources been considered?
- Can selected parallel work edit the project safely? Are edit conflicts handled through scheduling rather than fabricated business edges?

## Briefs and state

- Does the brief add confirmed repository findings worth handing over and directions supported by them?
- Is it omitted when there is nothing worth keeping? Is the Story complete without it, and will the executor still check the code?
- Does state keep current facts, Lanes, blockers, and acceptance without accumulating a history or copying stable rules?
- Do accepted Stories and Outcomes stay frozen? Do regression fixes point to the original result from a new Story?
- Is a repartition limited to entirely unstarted work, with every old Acceptance mapped completely and exactly once?

## Finish the review

Resolve mechanical diagnostics first. Then fix semantic findings and check their consequences. Finish when requirements, project facts, specifications, dependencies, and acceptance have no known conflict; relevant risk and non-success behavior are defined; no critical product decision is outstanding; and `nanospec validate` passes.

Record scope, version, findings, and anything still unverified. Do not copy this checklist into the result or invent a finding to justify a change. A specification review does not replace observing and accepting the actual delivery.
