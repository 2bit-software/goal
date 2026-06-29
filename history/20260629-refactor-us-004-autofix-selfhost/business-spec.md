# US-004 Run the autofixer across selfhost — Business Specification

## Overview

Step 2 of 3 of the self-host idiomatic effort. After the verbatim rename
(US-001..003), run `goal fix --inplace` across every `selfhost/*.goal` file so
that manual Go-style error propagation is mechanically idiomatized before the
per-package hand audits (US-005+). Because the compiler is now written in goal,
this also dogfoods `goal fix` on real compiler code: any case where the fixer
produces non-compiling output is a defect to be fixed in the fixer itself.

## Functional Requirements

### FR-1: Autofix is applied across selfhost
`goal fix --inplace` SHALL be run across all `selfhost/*.goal` files, and any
resulting source changes SHALL be committed.

### FR-2: The autofixer never emits non-compiling code
`goal fix` SHALL NOT rewrite a function's signature in a way that breaks its call
sites. Specifically, a `(T, error)` -> `Result[T, error]` conversion SHALL only
be applied when every call site of that function will remain valid after the
change. When it cannot prove call-site safety, the fixer SHALL leave the function
untouched and report a skip.

### FR-3: Fixed point
Re-running `goal fix` over selfhost after the autofix SHALL produce no further
file changes.

### FR-4: Gates stay green
After the autofix, `task check`, `task build`, `task fixpoint`, and the corpus
behavioral tier SHALL all pass.

## Acceptance Criteria

- [ ] `goal fix --inplace` has been run across all `selfhost/*.goal`; resulting
      changes (if any) are committed.
- [ ] Running `goal fix` over selfhost a second time reports/writes no further
      changes (idempotent fixed point).
- [ ] `task check` passes (includes the corpus transpile/behavioral/check tiers
      and the self-host port gates).
- [ ] `task build` passes.
- [ ] `task fixpoint` passes (goal-c-1 and goal-c-2 emit byte-identical Go).
- [ ] Existing `internal/fix` and `cmd/goal` fix tests continue to pass.

## User Interactions

CLI: `goal fix [-inplace] [path]`. Behavior unchanged except that the
`result-sig` rule is more conservative about which functions it converts.

## Error Handling

When a `(T, error)` function cannot be safely converted (exported, or referenced
somewhere `?` cannot apply), `goal fix` reports a `result-sig` skip to stderr
naming the function and the reason, and leaves the source unchanged. Operational
failures (bad path, unreadable/unwritable file) still fail the command.

## Out of Scope

- Manual enum/match/sealed/derive upgrades (US-005+).
- Per-package Result/? audits that coordinate cross-file call-site changes
  (US-005..US-012).
- Cross-file / package-aware call-site rewriting in `goal fix`.

## Open Questions

- None. The conservative-refusal behavior is the correct outcome for an
  automated `--inplace` pass; aggressive cross-file conversion is the realm of
  the per-package audits.
