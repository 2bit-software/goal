# US-019 eval assert at runtime

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop convention)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

The goscript tree-walking interpreter (internal/interp) must evaluate goal's
`assert` statement (ast.AssertStmt): a false condition panics with the located
assertion message; a true condition is a no-op. A unit test over a 10-assert
shaped program asserts the panic on a false assertion and normal completion on a
true one.
