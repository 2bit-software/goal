# Initiative: goal-fix-command

**Type**: feature
**Status**: in_progress
**Created**: 2026-06-25
**ID**: 6a3defb6-feature-goal-fix-command

## Steps

| Step | Profile | Status | Updated |
|------|---------|--------|--------|
| spec | feature | completed | 2026-06-25 20:30 |
| plan | plan | completed | 2026-06-25 20:40 |
| tasks | tasks | completed | 2026-06-25 20:50 |
| implement | implement | completed | 2026-06-25 21:30 |

## Description

Add a `goal fix` subcommand: a source-to-source migration aid that detects plain-Go
patterns inside `.goal` files and rewrites them into idiomatic goal (the inverse of the
lowering passes). Flagship transform: collapse manual error/nil propagation into the `?`
operator, and convert `(T, error)` signatures into `Result[T, error]`. Two modes: print
to stdout (default) and `-inplace` (write back). Built lexically on the existing
`scan`/`analyze`/`project` infrastructure — no AST.

## Goals

- Ship a body-local, ripple-free `?`-collapse fixer as the MVP (P1).
- Add `(T, error)` → `Result[T, error]` signature conversion with same-package call-site
  updates and exported-symbol warnings (P2).
- Catalog and (where body-local-safe) apply additional fixers, e.g. `switch`→`match` (P3).
- Never silently emit incorrect code: report every candidate not safely fixed; `goal
  check` remains the correctness authority.

## Progress

- 2026-06-25: spec step — research complete, business spec written, audited. Running in
  AutoMode (no branching, on `main`).
