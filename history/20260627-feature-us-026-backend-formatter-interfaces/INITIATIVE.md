# US-026 Add Backend and Formatter interfaces

**Type**: feature
**Created**: 2026-06-26
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs on base)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-26 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Phase 2 seam-laying story. Introduce the pluggable `Backend` and `Formatter`
interfaces so a new AST-driven engine can run behind a `--engine=ast` flag,
alongside the existing splice engine. Define a minimal `internal/sema.Info`
(fleshed out by US-027) so `Backend.Emit(*ast.File, *sema.Info) (Output, error)`
can be expressed. Prove the seam end-to-end: a goal file using no goal-specific
constructs transpiles through the new engine and the output compiles + vets via
the corpus behavioral tier.
