# Technical Requirements / Research — US-003

## Current state

- `selfhost/main.goal` imports `goal/internal/{backend,pipeline,project}`. Every
  ported `selfhost/*.goal` file also imports `goal/internal/{ast,backend,lexer,
  parser,pipeline,project,sema,token}`. So the emitted bootstrap compiler is
  actually built from the trusted `internal/*` Go (the fixpoint is trivially
  green — not a real self-host proof).
- The bootstrap (`Taskfile.yml`) emits `./selfhost` to `_bootstrap/sN` and runs
  `go build -o bin/goal-c-N ./_bootstrap/sN/selfhost` in the repo module, so
  `goal/internal/X` resolves to the real repo packages.

## Mechanics

- `project.Discover("./selfhost")` yields packages with module-relative `Dir`
  ("selfhost", "selfhost/backend", ...). Emit writes each to
  `join(emitDir, pkg.Dir)`, so the tree lands at `_bootstrap/sN/selfhost/*` with
  `main` at `_bootstrap/sN/selfhost`.
- To make `goal/selfhost/X` imports resolve to the emitted packages, the emit
  root needs its own `go.mod` (`module goal`) so `goal/selfhost/X` →
  `_bootstrap/sN/selfhost/X`. Build with `go build -C _bootstrap/sN -o
  ../../bin/goal-c-N ./selfhost` (a nested module; `-C` is needed because the
  path crosses into it). Go 1.26 supports `-C`. `_`-prefixed dirs and nested
  modules are excluded from `./...`, so `task check`/`task build` are unaffected.

## Plan

1. Rewrite `goal/internal/` → `goal/selfhost/` across all `selfhost/*.goal`
   (verified: every occurrence is an import line or a comment — no string
   literals — so the rewrite is safe). Update main.goal's doc comment.
2. Taskfile `bootstrap`: write a nested `go.mod` into each emit dir and build the
   stages with `go build -C`.
3. `internal/selfhost/port_test.go`: change all layout/deps/relDir keys from
   `internal/X` to `selfhost/X` (the selfhost sources now import
   `goal/selfhost/X`). `selfhost_test.go` is unchanged (it operates on the real
   `internal/*` sources).
4. `internal/selfhost/selfhost.go` `BuildAndTest`: rewrite `goal/internal/` →
   `goal/selfhost/` in the copied white-box test files so their sibling imports
   match the relocated package-under-test (the verbatim internal test files
   import `goal/internal/X`; their only non-import hit is a harmless comment).
   `BuildTranspiled` is unchanged (no test files; it writes sources verbatim).

## Verification

- `task check` (includes the self-host port gates + corpus transpile/behavioral/
  check tiers via the trusted compiler), `task build`, `task fixpoint`
  (now a genuine differential proof: goal-c-1/goal-c-2 built from the ported
  tree, byte-identical emit of ./selfhost).
