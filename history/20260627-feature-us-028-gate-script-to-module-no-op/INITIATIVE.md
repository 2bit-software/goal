# US-028 gate script-to-module no-op

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop branch — no per-story branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

Prove the script-to-binary upgrade path is frictionless: a sample goscript
program runs under the goscript interpreter and, UNCHANGED, transpiles and
builds as a Go+ module via the existing AST backend, producing equivalent
observable behavior. A gate test asserts the interpreter run and the
transpiled-then-built binary of the same source produce the same output — the
upgrade is a no-op.
