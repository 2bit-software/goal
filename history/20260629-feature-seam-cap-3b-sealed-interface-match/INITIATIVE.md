# seam-cap-3b-sealed-interface-match

**Type**: feature
**Created**: 2026-06-29
**Branch**: main (loop-runner: no branch, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-29 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

SEAM-CAP-3b: type-pattern `match` over a same-package sealed interface. Make
`match` work over a sealed-interface scrutinee (concrete type patterns like
`match n { *Ident => ..., *CallExpr => ... }`), distinct from enum variant
patterns. Build a sealed-interface implementor registry from `implements`
clauses; teach the parser to accept type-pattern arms; add a separate
sealedMatch backend lowering emitting a Go type-switch with `case *T:` labels;
enforce sema exhaustiveness against the registry. Fixed in BOTH internal/ Go
and selfhost/ .goal mirror. CAP-3 part 2 of 3 (same-package only; cross-package
is CAP-3c).
