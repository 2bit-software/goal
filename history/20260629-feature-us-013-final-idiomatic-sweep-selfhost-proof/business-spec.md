# US-013 Final idiomatic sweep and self-host proof — Business Specification

## Overview

This is the closing story of the SELF-HOST IDIOMATIC plan. After US-001..US-012
renamed the compiler into `selfhost/*.goal`, ran the autofixer, and audited each
package, this story performs a whole-compiler proof: that no auto-convertible plain-Go
error-propagation patterns remain anywhere in the selfhost tree, that any remaining
deliberately-Go constructs are documented, and that the idiomatic compiler still
self-hosts to a byte-identical fixpoint while passing the corpus. It closes the loop:
the compiler is written in idiomatic goal AND compiles itself.

## Functional Requirements

### FR-1: Whole-tree autofix reports zero auto-convertible sites
Running `goal fix` across the entire `selfhost/**/*.goal` tree SHALL produce no source
change (no auto-conversion). Advisory output (`skipped: [result-sig]` refusals and
`suggestion: [call-site]` notes) is permitted because neither rewrites source.

### FR-2: Deliberately-Go constructs are documented
Every construct that `goal fix` flags but does not convert SHALL be recorded in
DECISIONS.md with a rationale. Per-package refusals are already documented (US-005..
US-012); this story SHALL additionally document any file never covered by a per-package
audit (the top-level `selfhost/main.goal`).

### FR-3: Byte-identical self-host fixpoint
`task fixpoint` SHALL pass: goal-c-1 and goal-c-2 emit byte-identical Go for the
compiler's own source (`./selfhost`).

### FR-4: Goal-built compiler passes the corpus
The goal-built compiler SHALL pass the full corpus transpile + behavioral + check
tiers (exercised by `task check`).

## Acceptance Criteria

- [ ] `goal fix -inplace` over a copy of the whole selfhost tree yields an empty diff
      vs the original (zero auto-convertible propagation sites).
- [ ] The `goal fix` stderr report over the tree contains only `skipped`/`suggestion`
      lines and no `fixed` lines.
- [ ] Every flagged function maps to a documented refusal in DECISIONS.md, including a
      US-013 section covering `selfhost/main.goal` (`run`, `emitPackage`).
- [ ] `task check` is green (vet + full test suite, incl. corpus + selfhost port gates).
- [ ] `task build` is green.
- [ ] `task fixpoint` prints `FIXPOINT OK` (goal-c-1 and goal-c-2 byte-identical).

## User Interactions

Developer-facing only: `goal fix <path>`, `task check`, `task build`, `task fixpoint`.

## Error Handling

If `goal fix` were to convert anything (a non-empty diff or a `fixed` line), or if any
flagged construct were undocumented, the story is not complete and the gap must be
closed (convert or document). If the fixpoint diff is non-empty, the self-host proof
fails and must be diagnosed before the story passes.

## Out of Scope

- New idiomatic conversions beyond documenting/finishing leftovers from US-005..US-012.
- Cross-package switch->match sealing refused in prior stories (explicitly out of scope
  per the US-007 §9 switch-coexistence decision).
- Any change to corpus fixtures or the Go-side `internal/*` packages.

## Open Questions

- None. Reconnaissance confirms the tree is already at the autofix fixed point; the only
  remaining work is documenting `main.goal`'s two bare-error refusals and proving the
  gates.
