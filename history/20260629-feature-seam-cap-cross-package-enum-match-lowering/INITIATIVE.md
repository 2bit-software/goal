# SEAM-CAP: cross-package enum-match lowering in the backend

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

Make the goal backend lower a `match` over an enum DEFINED in an imported
package. Today only Result/Option cross package boundaries (hardcoded
special-casing); a user enum imported from another package fails to lower with
`backend: unsupported statement-position match on ""`. Fix `matchQualifier` to
resolve a package-qualified variant pattern (a `SelectorExpr` like
`ast.FuncMod.FuncDerive`), and make enum resolution find enums declared in
imported packages via cross-package sema.Info (generalizing beyond Result/Option).
Add a regression fixture proving a cross-package enum match transpiles and
behaves identically to the equivalent switch.
