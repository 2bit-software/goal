# US-022 Gate interp on native sema only

**Type**: feature
**Created**: 2026-06-28
**Branch**: ralph/ast-frontend-rewrite (loop runs linear on base; no per-story branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-28 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

The goscript interpreter's run path must validate input solely through
internal/sema (native AST checks) and refuse a program that fails a sema
guarantee BEFORE evaluation, with a located diagnostic. A dependency test
asserts internal/interp does not depend on internal/typecheck or go/types
(the go/types-over-lowered-Go crutch is fine for the Go-transpile path but
NOT for the interpreter — REWRITE-ARCHITECTURE.md §3.2).
