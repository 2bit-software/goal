# Technical Requirements / Research — US-004

## How `goal fix` works (relevant to this story)

- `cmd/goal` `fix` subcommand: `goal fix [-inplace] [path]`. For a directory it
  `project.Discover`s packages and calls `fix.File(src)` on each file
  INDEPENDENTLY (no cross-file coordination). With `-inplace` it writes changed
  files back; otherwise it prints the rewrite to stdout. Reports go to stderr.
- `internal/fix` rules (`fix.File`, fixed-point loop):
  - `result-sig` (resultsig.go): converts a `(T, error)` function into
    `Result[T, error]`, rewriting `return v, nil` -> `return Result.Ok(v)`.
  - `propagate` (propagate.go): collapses `v, err := g(); if err != nil { return ... }`
    into `v := g()?` INSIDE a Result/Option function.
  - `match` (match.go): reports switch-over-enum (no rewrite).
  - `reportCallSites` (callsite.go): suggests sites still needing a signature change.

## Dogfooding bug found by a dry run on selfhost

`goal fix --inplace` over a scratch copy of selfhost converts exactly three
functions, and ALL THREE break compilation, because `result-sig` converts a
function's call ABI from a 2-tuple to a single `Result` but `fix.File` (per file)
cannot rewrite the call sites it does not control:

- `selfhost/project/project.go` `Discover` — EXPORTED; caller
  `selfhost/main.goal:64` (`pkgs, err := project.Discover(root)`) is in another
  file/package and is never updated.
- `selfhost/sema/analyze.goal` `Analyze` — EXPORTED; same class.
- `selfhost/sema/foreign.goal` `goListResolve` — unexported, but its same-file
  caller `DefaultResolver` tail-returns it (`return goListResolve(...)`) and is
  itself not converted (DefaultResolver is exported and has a non-propagating
  return), so it now returns a `Result` from a `(string, error)` function.

## Fix (in internal/fix)

Make `result-sig` conservative about call-site safety, matching the package's
stated philosophy ("never emit incorrect code"):

1. Refuse to convert EXPORTED functions — their callers may live in other
   files/packages `fix.File` never sees. (Report a Skip that still names the
   function as exported.)
2. Refuse to convert a function if any reference to it in the file is NOT a
   collapsible error-propagation call site (`v, err := f(...)` immediately
   followed by an `if err != nil` propagation guard inside a function that is or
   will be Result/Option-returning). This rejects tail returns
   (`return f(...)`), unguarded calls, and value/argument uses — the shapes
   `?` cannot replace.

These two rules close all three selfhost miscompiles while preserving every
existing fix test (the test functions are either unexported with no in-file
callers, or have exactly the collapsible-propagation call site). The honest net
result on selfhost is zero safe conversions: every fallible function bottoms out
at an exported cross-package API, which the per-package audits (US-005+) convert
with cross-file coordination. So selfhost source is unchanged, the gates stay
green, and `goal fix` is idempotent.
