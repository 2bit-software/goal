# Initiative: fix-statement-context-error-passthrough

**Type**: feature
**Status**: complete
**Created**: 2026-06-25
**ID**: 6a3dfdf6-feature-fix-statement-context-error-passthrough

## Steps

| Step | Profile | Status | Updated |
|------|---------|--------|--------|
| spec | feature | complete | 2026-06-25 21:30 |
| plan | plan | complete | 2026-06-25 21:30 |
| tasks | tasks | complete | 2026-06-25 21:30 |
| implement | implement | complete | 2026-06-25 21:30 |

## Description

Extend `goal fix` to convert the statement-context error guard
`if err := doThing(); err != nil { return Result.Err(err) }` into `doThing()?` when the call's
only output is the error, inside a `Result[T, error]` function. This closes the fixer's blind
spot: it previously collapsed only the value-binding form (`x, err := f(); if err != nil …`),
silently leaving the if-init form untouched even when sitting two lines below a successful
collapse (the `state/state.goal` case in the bug report).

## Goals

- Collapse pure error passthroughs written as if-init guards to `?` (the 6 missed cases).
- Preserve the fixer's never-change-behavior invariant: wrapped / decorated / non-zero /
  commented / else-bearing guards, and guards in non-Result functions, are left untouched.

## Progress

- Ran in AutoMode on `main` (no branching, per request).
- Added `fixPropagateInit` plus `ifBodyBrace` / `topLevelSemicolon` helpers to
  `internal/fix/propagate.go`; wired it into `fix.File`.
- Reused the existing `validPropagationReturn` body check, so the safety contract is shared
  with the value-binding rule.
- Added 5 unit tests to `internal/fix/fix_test.go` (happy path + idempotence, wrapped error,
  comment Skip, else, non-Result function).
- Verified end-to-end with the `goal fix` CLI on a mixed value-binding + init-guard file —
  all three collapse to `?`. Full `go test ./...` suite green.
