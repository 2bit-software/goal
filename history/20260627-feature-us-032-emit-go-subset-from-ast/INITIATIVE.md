# us-032-emit-go-subset-from-ast

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop constraint: no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

US-032: Emit the Go subset from AST. Extend internal/backend's AST Go emitter
(emit.go) to cover the full ordinary-Go subset that goal source uses — notably
expression `switch`/`case` statements, which the US-026 seed emitter omits — so
the backend walks the AST and emits Go source text for the whole Go subset,
formatted once via the Formatter. A goal file using only plain-Go constructs
transpiles through the new backend and passes the behavioral tier (temp-module
`go build` + `go vet`).
