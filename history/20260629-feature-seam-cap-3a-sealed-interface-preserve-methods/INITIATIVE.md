# SEAM-CAP-3a: Sealed interfaces preserve method signatures

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

A goal `sealed interface` must keep its declared method signatures in emitted Go
(e.g. `type Node interface { Pos() Position; End() Position; isNode() }`), not just
the marker method. Today `genSealedInterface` emits only `type Name interface{ isName() }`
and `sealedInterfaceDecl` discards the parsed interface body, so a sealed interface's
declared methods silently vanish — which would make sealing ast.Node impossible.

Fix in BOTH internal/ (lower.go / emit.go) AND selfhost/ (.goal mirror). Scope:
method-signature preservation ONLY. The implementor registry, match grammar,
exhaustiveness, and cross-package propagation are CAP-3b/3c — out of scope here.
