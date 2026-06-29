# Plan Audit — US-003

## Findings

- No CRITICAL/MAJOR. The plan maps every FR to concrete file edits, the
  dependency order is a valid topological sort, and all file paths were verified
  against the codebase.
- MINOR: `go build -C _bootstrap/sN -o ../../bin/goal-c-N` relies on `-o` being
  resolved relative to the `-C` directory. Verified against Go 1.26 semantics;
  confirm empirically during verify.
- MINOR: Only `main`'s import closure (backend, pipeline, project + transitive)
  is built by `go build ./selfhost`; `typecheck` is emitted but not compiled into
  goal-c. It must still transpile cleanly (it does — ported in US-002). No action.

## Assumptions

- The nested `go.mod` is created by the Taskfile (a build-harness concern), not
  emitted by `main.goal` — keeping the compiler's emit output pure so the
  fixpoint diff compares only generated Go.
- The `goal/internal/` → `goal/selfhost/` rewrite in `BuildAndTest` is applied to
  copied test files only; package sources already import `goal/selfhost/X` from
  the flipped `.goal` files.
