# Tasks — US-007 Idiomatic audit: ast

## Task 1: Confirm package facts (foundation, no deps) — completed
- Confirm `selfhost/ast` has no error-returning functions and the only
  node-kind type-switch is `Walk` over the plain `Node` interface.
- Confirm FuncMod/ChanDir are `type X int` + iota consumed cross-package via
  `==`/plain-switch in sema/backend/parser.
- Confirm `goal fix selfhost/ast/*.goal` produces no diff and no report.
- Spec coverage: FR-1, FR-2, FR-3.
- Files: (read-only) selfhost/ast/*.goal, selfhost/{sema,backend,parser}/*.goal.

## Task 2: Record the audit decision (depends on Task 1) — completed
- Append a "self-host idiomatic audit — US-007 (ast)" section to DECISIONS.md
  recording the three refusals (category interfaces -> sealed interface;
  FuncMod/ChanDir -> enum; Walk type-switch -> match) and the no-Result/?
  finding, each with reason and an "Over" alternatives note, matching the
  US-005/US-006 format.
- Spec coverage: FR-1, FR-2, FR-3.
- Files: DECISIONS.md.

## Task 3: Verify (depends on Task 2) — completed
- `goal fix selfhost/ast/*.goal` -> no diff/report (AC-2).
- `task check` -> green (port gate + ast tests + go vet).
- `task build` -> green.
- `task fixpoint` -> FIXPOINT OK (byte-identical).
- Spec coverage: FR-3, FR-4, all ACs.
- Files: none (verification only).
