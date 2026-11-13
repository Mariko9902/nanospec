# Outcome Record Template

`acceptance/outcome.md` freezes the observed result and evidence for every Acceptance item. It records what was accepted without rewriting the Story.

During execution, `nanospec evidence` maintains the matching `acceptance/evidence.md` record and `nanospec verify` records review. Once the acceptance conditions are met, `nanospec accept` freezes it as Outcome. Keep one record, not two copies.

Write observations and explanations in the Cycle's `spec_language`. Fixed headings, fields, and enums stay unchanged. Commands and raw results retain their actual text. A later language change does not translate a frozen record.

```md
---
story: story-01-<slug>
spec_rev: rev-<N>
story_digest: sha256:<64-lowercase-hex-digits-for-story.md>
implemented_by: <implementer-id>
verified_by: <verifier-id>
verification_mode: self | independent
verified_at: <ISO-8601>
verified_acceptance: [A1, A2]
first_pass: true | false
rework_count: <nonnegative-integer>
spec_change_count: <nonnegative-integer>
accepted_by: <acceptor-id>
accepted_at: <ISO-8601>
acceptance_method: human | verified
---

## Acceptance Results

Verification Context: {"spec_revision":"rev-<N>","code_revision":"<full-Git-commit-or-sha256:digest>","environment":"<entry>","observed_at":"<ISO-8601>"}

- A1: confirmed
  - Observed: <The actual business result.>
  - Evidence Command: <The actual command, or none: manual observation.>
  - Evidence Result: <The actual command or observation result.>
  - Evidence Artifacts: <A page, response, export, snapshot, log, or report path.>
  - Evidence Context: {"spec_revision":"rev-<N>","code_revision":"<full-Git-commit-or-sha256:digest>","environment":"<entry>","observed_at":"<ISO-8601>"}
- A2: confirmed
  - Observed: <The actual business result.>
  - Evidence Command: <The actual command, or none: manual observation.>
  - Evidence Result: <The actual result.>
  - Evidence Artifacts: <Artifact path.>
  - Evidence Context: {"spec_revision":"rev-<N>","code_revision":"<full-Git-commit-or-sha256:digest>","environment":"<entry>","observed_at":"<ISO-8601>"}
```

## Record rules

- `spec_rev` records the accepted Cycle revision. `story_digest` binds the exact accepted Story contents. The acceptance command writes both and detects changes to the Story after acceptance.
- `implemented_by`, `verified_by`, and `accepted_by` identify different responsibilities. Keep all three fields even when their values match.
- `self` verification requires the implementer and verifier to match. `independent` requires different identities. `verified_acceptance` must list every Acceptance ID exactly once.
- `human` acceptance records the person who made the decision. `verified` records the authorized executor who checked the conditions and accepted the result, not the identity of the person who granted authority.
- `rework_count` counts returns to implementation after a complete review finds that implementation or evidence fails Acceptance. Several problems in one review count as one round. Rejecting a ready result also adds one round.
- `first_pass` is true exactly when `rework_count` is zero.
- `spec_change_count` comes from the Lane's `Spec Changes`. A revision adds one when execution meaning affecting this Story changes after work has started. Progress, evidence, wording, and generated-view updates do not count.
- These counts help compare planning and verification methods. They do not measure elapsed time or rate an executor.
- `verified` acceptance requires the Story's explicit authority, scope, and coverage review, plus independent verification and all acceptance conditions. A person may still accept such a Story with the `human` method.
- Keep the Story and optional brief in the Story root. Records go in `acceptance/`; raw material goes in `acceptance/evidence/`.
- `evidence.md` exists during implementation and pending acceptance. It becomes `outcome.md` on acceptance. They never coexist.
- Artifact paths are relative to `acceptance/`, such as `evidence/screenshots/detail.png`. Shared material can point to the owning Story's `acceptance/evidence/`. Moving an artifact does not change the command, version, environment, or observed fact.
- Acceptance Results covers every Story Acceptance ID. Each item keeps `Observed` and all four evidence fields shown above, including its own `Evidence Context`.
- For a manual observation, write `none: manual observation`; do not invent a command.
- Context values are JSON objects containing exactly `spec_revision`, `code_revision`, `environment`, and `observed_at`. The code reference is a full 40-digit Git commit or a `sha256:` digest with 64 hexadecimal digits. Environment identifies the actual entry or working directory; the observation time includes its timezone. New observations use the prepared revision. Unaffected observations retain the revision and context under which they were actually captured.
- `Verification Context` summarizes the latest recorded observation. Every item keeps its own complete `Evidence Context`, including when its values match the summary. Do not refer to the summary or another item instead of recording those values. Updating the summary never changes an existing item's context. The code reference must identify what actually ran; describe relevant uncommitted changes or an artifact digest when a base commit alone is insufficient.
- A report, page, or data result can serve as evidence when it connects the item's input, expectation, and actual result. A missing result or a bare "passed" does not show what was observed.
- Keep accepted records frozen. Correct a recording error only as such. New requirements, omissions, regressions, and later risks belong in subsequent Stories. A past result does not automatically prove that the current system still meets the same condition.
