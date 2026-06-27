# US-010 archive legacy per-feature transpilers

**Type**: refactor
**Created**: 2026-06-26
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| scaffold | done | 2026-06-26 |
| verify | done | 2026-06-26 |
| cutover | done | 2026-06-26 |
| cleanup | done | 2026-06-26 |
| done | in_progress | 2026-06-26 |

## Cleanup note

Pure relocation: no duplicated old code existed, so cleanup removed nothing.
The 11 `features/NN/transpiler/` modules now live under
`attic/features/NN/transpiler/`. Root module `go build/vet/test ./...` green
before and after (these modules were always excluded from module `goal`).

## Description

Archive the superseded standalone per-feature transpilers so they stop implying
they are live. The `features/NN/transpiler/` directories are separate Go modules
(each has its own `go.mod`) excluded from the root `goal` module. Move them under
`attic/` and ensure no `features/*/transpiler` path remains. `go build ./...` and
`go test ./...` for the root module must remain green (they never included these
modules).
