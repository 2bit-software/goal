# SEAM-CAP-3c — Capability: cross-.goal-package sealed-interface match

**Type**: feature
**Created**: 2026-06-29
**Branch**: main (loop runner — no branch, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-29 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

CAP-3 part 3 of 3. Extend the SEAM-CAP-2 goal-source foreign-enrichment path so a
sibling .goal package's sealed-interface implementor set (its `implements` clauses)
is projected into a consumer's `sema.Info.Sealed` + `sema.Info.SealedImpls`
(keyed `alias.Type`, implementors qualified `*alias.T`). This lets a cross-package
type-pattern `match` resolve and exhaustiveness-check in the real
`goal build ./selfhost` topology — the final prerequisite for SEAM-004 (35 of its
36 type-switches consume ast.Node/Expr/Stmt/Decl from OTHER .goal packages).

Mirror line-for-line in BOTH internal/sema/foreign.go and selfhost/sema/foreign.goal.
Backend sealedMatch lowering is already pattern-shape driven, so it already lowers
cross-package; this story closes only the sema resolution/exhaustiveness gap.
