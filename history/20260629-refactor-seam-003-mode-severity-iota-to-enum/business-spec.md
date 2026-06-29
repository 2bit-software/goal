# SEAM-003: sema.Mode and sema.Severity iota -> goal enum — Business Specification

## Overview

The self-hosted goal compiler classifies functions by error-propagation shape
(`Mode`) and diagnostics by enforcement strength (`Severity`). Both are currently
Go-style `iota` integer constants. This change expresses them as idiomatic goal
`enum`s and converts every consumer across the `sema`, `backend`, and `typecheck`
packages to goal's exhaustive `match`, so the compiler's own source demonstrates
the language feature it implements. The change is behavior-preserving: the same
diagnostics and the same generated Go are produced for all programs.

## Functional Requirements

### FR-1: Mode is a goal enum
`sema.Mode` SHALL be a goal `enum` with variants ModeNone, ModeResult,
ModeResultClosed, ModeOption — not `type Mode int` + iota.

### FR-2: Severity is a goal enum
`sema.Severity` SHALL be a goal `enum` with variants Error, Warning — not
`type Severity int` + iota. Its `String()` method SHALL render "warning" for
Warning and "error" otherwise, identical to today.

### FR-3: All consumers converted atomically
Every cross- and same-package consumer SHALL be updated in the same change:
==/!=/plain-switch over Mode or Severity SHALL become `match`; bare variant
references SHALL become `Enum.Variant` (same package) or `pkg.Enum.Variant`
(cross package). No plain `switch` over either enum SHALL remain.

### FR-4: Enum zero values set explicitly
Every constructor of a struct holding a Mode or Severity field, and every
function returning such a value, SHALL set the enum field explicitly. A goal
enum's zero value is nil (not the first variant), so any implicit zero would
fault a later `match` at runtime.

### FR-5: No residual numeric/ordering dependence
No member of either enum SHALL be retained as iota. There is no array-index,
range-bound, or wire-serialization use of Mode or Severity, so the conversion is
total.

## Acceptance Criteria

- [ ] `task check` is green (after any golden regen).
- [ ] `task build` is green.
- [ ] `task fixpoint` reports FIXPOINT OK (stage1 == stage2 on the new source).
- [ ] The corpus behavioral tier is unchanged.
- [ ] No plain `switch` and no `==`/`!=` over a Mode or Severity value remains in
      selfhost/{sema,backend,typecheck}.
- [ ] DECISIONS.md records the conversion, superseding the US-011 "Mode and
      Severity stay iota" refusal with the seam rationale.

## User Interactions

None — internal compiler refactor. No CLI, API, or user-visible surface changes.

## Error Handling

Diagnostics emitted by the checker are unchanged in code, severity, and message.
The Severity `String()` rendering is preserved exactly.

## Out of Scope

- token.Kind (SEAM-002 deliberately kept it iota for numeric identity).
- AST interface sealing and type-switch conversion (SEAM-004).
- Result/? API lifting (SEAM-005).

## Open Questions

None. The conversion pattern, the consumer sites, and the nil-fault hazards are
fully enumerated in technical-requirements-research.md from a completed audit and
the proven SEAM-002 precedent.
