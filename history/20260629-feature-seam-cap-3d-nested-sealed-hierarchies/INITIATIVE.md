# SEAM-CAP-3d: nested sealed-interface hierarchies

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

When a sealed interface B embeds a sealed interface A, an implementor declared
`implements B` must register as an implementor of BOTH A and B and emit BOTH
markers (isB() and isA()) so the emitted Go compiles and exhaustiveness works at
both levels. Implemented via an embedding cascade (no parser change). Fix in both
internal/ Go and selfhost/ .goal. Final CAP-3 prerequisite before SEAM-004 can
seal the 2-level AST (Expr/Stmt/Decl/Spec embed Node).
