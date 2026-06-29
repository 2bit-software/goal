# go->goal Upgrade Skill — Business Specification

## Overview

A repeatable skill that upgrades an existing Go codebase to idiomatic goal,
scoped to a SINGLE go.mod package OR a SINGLE .go file. It packages the entire
idiomatization pipeline proven across SEAM-002..006 into a one-command tool any
goal adopter can run in any Go repo. It performs both the mechanical autofixer
pass (`goal fix`) and the manual idiom transforms the autofixer does not do
(iota->enum, type-switch->sealed+match, method-on-enum->free label fn, fallible
API->Result/?), then verifies the result still builds and reports converted vs
documented non-fit.

## Functional Requirements

### FR-1: Scope guard
The skill SHALL accept either one `.go` file or one `go.mod` package directory as
its scope, and SHALL warn/refuse when pointed at a whole multi-package module
(more than one package directory under the target).

### FR-2: Rename
The skill SHALL rename `.go` -> `.goal` within scope, handling the package
clause, build tags, and reserved-word identifier collisions (`match`, `enum`,
`assert`), exactly as the selfhost rename stories did.

### FR-3: Autofix
The skill SHALL run `goal fix` to mechanically convert `(T,error)` + manual
`if err != nil` propagation into `Result`/`?`, and SHALL report what it changed
and what it skipped.

### FR-4: Manual idioms
The skill SHALL apply the non-autofixer upgrades catalogued from SEAM-002..005:
iota const block -> goal `enum` (unless numeric-identity/wire/ordering
dependence); type-switch over a closed/sealed scrutinee -> sealed interface +
exhaustive `match`; method on a would-be-enum type -> free label function;
exported fallible `(T,error)` -> `Result`/`?` where pure-propagation. It SHALL
record each genuine refusal with its reason (numeric identity, accumulator,
multi-value, comma-ok, cross-package consumer not in scope).

### FR-5: Verify + report
The skill SHALL confirm the upgraded scope still transpiles/builds and SHALL emit
a DECISIONS-style summary of what converted vs what was left as a documented
non-fit.

### FR-6: Carry-forward guidance
The skill SHALL encode the carry-forward gotchas as guidance it emits/checks:
enum/sealed zero value is nil (set fields explicitly); enums can't carry methods;
value-position match lowers only as `:=` / `var x T = match` / `return match`;
cross-package idioms rely on whole-program enum/sealed-fact propagation.

## Acceptance Criteria

- [ ] A skill exists in the project's skill location, accepts EITHER one `.go`
      file OR one `go.mod` package dir, and refuses/warns on a multi-package module.
- [ ] Rename step handles package clause, build tags, and reserved-word collisions.
- [ ] Autofix step runs `goal fix` and reports changed vs skipped.
- [ ] Manual-idiom step applies the four catalogued transforms and records each
      refusal with a reason from the documented taxonomy.
- [ ] Verify step confirms the upgraded scope builds and emits a converted-vs-
      non-fit summary.
- [ ] The skill is documented (usage, scope rules, idiom catalogue) and dogfooded
      on at least one real example proving buildable idiomatic goal.
- [ ] `task check`, `task build`, `task fixpoint` remain green.

## User Interactions

A user (or agent) invokes the skill, pointing it at one `.go` file or one package
directory. The skill walks the five steps in order, emits per-step reports, and
finishes with a DECISIONS-style summary. The dogfood example is runnable and
documented so a reader can reproduce buildable idiomatic goal output.

## Error Handling

- Multi-package module target: warn/refuse with the scope rule explained.
- Non-existent / wrong-extension target: clear error.
- A transform that does not fit: not an error — recorded as a documented non-fit
  with its reason, and the file is left in its safe (pre-transform) state for
  that construct.
- Dogfood runs on a COPY; tree source the project depends on is never destroyed.

## Out of Scope

- Upgrading a whole multi-package module in one shot (the unit is one file or one
  package, matching the per-package fix model).
- Changing the `goal` compiler or `goal fix` itself (those shipped in SEAM-CAP..006).
- Converting cross-package consumers that live outside the chosen scope (recorded
  as a non-fit, not converted).

## Open Questions

- None blocking. Skill location resolved to `.claude/skills/` (standard Claude
  Code project skill dir; repo has no pre-existing project skill).
