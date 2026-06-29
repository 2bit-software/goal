# US-003 Support generic function declarations

**Type**: feature
**Created**: 2026-06-29
**Branch**: main (no branch — loop linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-29 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Top-level generic function declarations must parse and transpile so the
language can express any compiler code without restriction.
`func Identity[T any](x T) T { return x }` must parse without
`expected (, found [`, ast.FuncType must carry a TypeParams field, the
backend must emit the type-parameter list, and the generic function must
transpile to valid Go that `go build` accepts (including a constrained
param like `[T comparable]`).
