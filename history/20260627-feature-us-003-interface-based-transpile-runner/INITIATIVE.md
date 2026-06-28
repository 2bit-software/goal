# US-003 Build interface-based transpile runner

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27T02:08Z |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

As a compiler engineer, I need a transpile runner behind an interface so
multiple front-ends can be judged identically. internal/corpus defines a
`Transpiler` interface `{ Transpile(src string) (pipeline.Output, error) }`
and a runner that gofmt-normalizes both got and want before comparing. A test
runs every transpile case in the manifest against pipeline.Transpile and all
pass.
