# Technical Requirements & Research — US-004

## CLI contract (must honor)

The bootstrap drives the existing `goal build --emit=<dir> <path>` contract
(`cmd/goal/main.go`: `cmdBuild`, `emitFiles`, `parseFlags`). `--emit=<dir>`
writes generated Go to `filepath.Join(dir, pkg.Dir)/<name>.go`. The goal-written
skeleton must replicate this same layout so its emit is interchangeable.

## Skeleton design (`selfhost/main.goal`)

A `package main` goal program that is a thin `goal build --emit` equivalent:
parse `build --emit=<dir> <path>`, `project.Discover(path)`,
`backend.TranspilePackage(pkg)` for each package, and write `out.Files` (+
`out.Tests`) under `filepath.Join(emitDir, pkg.Dir)`. Imports
`goal/internal/{project,backend,pipeline}` plus `fmt`, `os`, `path/filepath`,
`strings` — all pass through the goal front-end as foreign Go. The skeleton
imports the real `internal/*` packages; US-005+ replace those with ported ones.

## Bootstrap mechanics (SELF-HOST-RESEARCH.md §5)

```
stage 0: task build                                   # trusted Go-built ./bin/goal
stage 1: ./bin/goal   build --emit=_bootstrap/s1 ./selfhost; go build -o bin/goal-c-1 ./_bootstrap/s1/selfhost
stage 2: ./bin/goal-c-1 build --emit=_bootstrap/s2 ./selfhost; go build -o bin/goal-c-2 ./_bootstrap/s2/selfhost
fixpoint: goal-c-1 and goal-c-2 each emit Go for ./selfhost; diff -r must match
```

## Key constraint: keep `task check` / `task build` green

The Go toolchain (`go build ./...`, `go vet ./...`, `go test ./...`) ignores
directories whose names begin with `_` or `.` when expanding `./...`, but an
explicit path like `go build ./_bootstrap/s1/selfhost` still builds. So emit all
bootstrap artifacts under a `_bootstrap/` directory (verified empirically) and
keep the goal-c binaries in `bin/` (already gitignored). Add `_bootstrap/` and
`bin/goal-c-*` to `.gitignore`. `selfhost/` holds only `.goal` files, which the
Go toolchain ignores entirely.

## Fixpoint determinism

goal-c-1 and goal-c-2 both link the real `internal/backend`, so for the skeleton
they are functionally identical transpilers and their emitted Go for `./selfhost`
is byte-identical. As US-005+ port packages into `selfhost/`, the fixpoint
becomes a genuine differential proof.
