# us-033-lower-emit-enums-implements

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop runs on base branch, no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

US-033: Lower and emit enums and implements on the new AST backend so closed
types work. The Go emitter (internal/backend) must produce the §8.1 sum encoding
for enums and the assertion/marker for implements, consuming sema.Info. The
01-enums and 07-implements transpile cases must pass the behavioral tier
(corpus.RunCompile: go build + go vet) through the new --engine=ast backend.
