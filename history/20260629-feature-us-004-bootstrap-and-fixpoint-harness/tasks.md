# Tasks — US-004 Bootstrap and fixpoint harness

Status: Task 1 completed, Task 2 completed, Task 3 completed.

## Task 1: Add the goal-written compiler skeleton
- **Files**: `selfhost/main.goal`
- **Covers**: FR-1
- **Do**: Create a `package main` goal program that is a `goal build --emit`
  equivalent — parse `build --emit=<dir> <path>`, `project.Discover(path)`,
  `backend.TranspilePackage(pkg)` per package, write `out.Files` + `out.Tests`
  under `filepath.Join(emitDir, pkg.Dir)`. Imports `goal/internal/{project,
  backend,pipeline}`, `fmt`, `os`, `path/filepath`, `strings`.
- **Verify**: `./bin/goal build --emit=_bootstrap/s1 ./selfhost` (after
  `task build`) emits `_bootstrap/s1/selfhost/main.go`, and
  `go build -o bin/goal-c-1 ./_bootstrap/s1/selfhost` compiles.

## Task 2: Add bootstrap and fixpoint Taskfile targets
- **Files**: `Taskfile.yml`
- **Covers**: FR-2, FR-3
- **Do**: Add `bootstrap` (stage-0 -> goal-c-1 -> goal-c-2) and `fixpoint`
  (`deps: [bootstrap]`, emit fa/fb, `diff -r`, echo FIXPOINT OK) targets.
- **Verify**: `task bootstrap` exits 0; `task fixpoint` exits 0.

## Task 3: Ignore build artifacts
- **Files**: `.gitignore`
- **Covers**: FR-4
- **Do**: Ignore `_bootstrap/` and `bin/goal-c-*`.
- **Verify**: `git status` shows no untracked bootstrap artifacts after running
  the targets; `task check` and `task build` green.
