# Implementation Tasks — US-003

## Task 1: Flip selfhost source imports to goal/selfhost/*
**Status**: completed
**Files**: `selfhost/main.goal`, `selfhost/**/*.goal` (token, lexer, ast, parser, sema, project, pipeline, backend, typecheck)
**Depends on**: (none)
**Spec coverage**: FR-1; AC #1, #2
**Verify**: `grep -rn 'goal/internal/' selfhost/` returns nothing.

### Instructions
- Replace every `goal/internal/` with `goal/selfhost/` across all `selfhost/*.goal`
  files (imports + the one doc comment in `main.goal`). Verified safe: no string
  literals contain the path. Foreign `go/*` and stdlib imports are untouched.

## Task 2: Rewrite copied test-file imports in BuildAndTest
**Status**: completed
**Files**: `internal/selfhost/selfhost.go`
**Depends on**: (none)
**Spec coverage**: FR-4 (keeps port behavioral gate green after the relocation)
**Verify**: compiles under `go vet ./internal/selfhost`.

### Instructions
- In `BuildAndTest`, after `os.ReadFile(tf)`, replace `goal/internal/` with
  `goal/selfhost/` in the bytes before writing into the temp module, so the
  white-box test files' sibling imports resolve against the relocated
  (selfhost-keyed) package-under-test and its deps. Add a clarifying comment.
  `BuildTranspiled` stays untouched.

## Task 3: Flip port_test.go layout/deps/relDir keys to selfhost/*
**Status**: completed
**Files**: `internal/selfhost/port_test.go`
**Depends on**: Task 1, Task 2
**Spec coverage**: FR-4
**Verify**: `go test ./internal/selfhost` passes.

### Instructions
- Change every `BuildTranspiled` layout key, every `deps` map key, and every
  `BuildAndTest` `relDir` from `internal/<pkg>` to `selfhost/<pkg>`. Leave
  `selfhost_test.go` unchanged (it operates on real `internal/*` sources).

## Task 4: Bootstrap from the self-contained tree
**Status**: completed
**Files**: `Taskfile.yml`
**Depends on**: Task 1
**Spec coverage**: FR-2, FR-3; AC #3
**Verify**: `task fixpoint` exits 0 (FIXPOINT OK).

### Instructions
- In `bootstrap`, after each `--emit=_bootstrap/sN ./selfhost`, write a nested
  `go.mod` (`module goal\n\ngo 1.26\n`) into `_bootstrap/sN`, then build with
  `go build -C _bootstrap/sN -o ../../bin/goal-c-N ./selfhost` (replacing the
  old `go build -o bin/goal-c-N ./_bootstrap/sN/selfhost`). `fixpoint` unchanged.

## Task 5: Full verification
**Status**: completed
**Files**: (none)
**Depends on**: Task 1-4
**Spec coverage**: FR-1..FR-4; AC #1-#5
**Verify**: `task check` && `task build` && `task fixpoint` all green; AC grep empty.
