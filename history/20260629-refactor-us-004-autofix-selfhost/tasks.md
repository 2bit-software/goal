# Implementation Tasks

## Task 1: Make `result-sig` refuse unsafe conversions
**Status**: completed
**Files**: `internal/fix/resultsig.go`
**Depends on**: (none)
**Spec coverage**: FR-2; AC-6
**Verify**: `go test ./internal/fix ./cmd/goal`

### Instructions
- Split per-decl structural analysis into `classifyResultSig` returning either a
  `*sigCand` (convertible candidate) or a near-miss `*Report` (bare error,
  multi-value, non-propagating return) — preserving the exact existing Skip
  messages so `TestDecoratedErrorNotConverted` / `TestMultiValueNotConverted`
  still pass.
- In `fixResultSig`: collect candidates, compute `successTOf` and `willResult`,
  compute the safe call-site offset set via `safeLocalPropagationCalls`, then for
  each candidate apply the safety gate:
  - exported -> Skip "exported `X` has callers fix cannot see; not auto-converted to Result"
    (keeps `TestFixExportedWarning` green: message still contains "exported" + name).
  - any unsafe reference (`unsafeRefOffset`) -> Skip "`X` is called where `?`
    cannot apply (e.g. a tail return or unguarded call); not auto-converted to Result".
  - else emit the signature replacement + success replacements + record the Change.
- Remove the old exported `Warn`-and-convert branch.
- Add helpers `safeLocalPropagationCalls` (mirror fixPropagate's match conditions,
  recording local-call Fun ident offsets) and `unsafeRefOffset` (ast.Walk for
  Idents named X, excluding the decl, flagging any offset not in the safe set).

## Task 2: Run the autofixer across selfhost and verify idempotence
**Status**: completed
**Files**: `selfhost/**/*.goal` (only if the corrected fixer changes them)
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-3; AC-1, AC-2
**Verify**: rebuild `goal`; `goal fix --inplace selfhost`; re-run `goal fix selfhost`
(no further changes); `git status` on `selfhost/` reflects exactly the autofix output.

### Instructions
- `go build -o bin/goal ./cmd/goal`.
- `./bin/goal fix --inplace selfhost` (reports to stderr, writes changed files).
- Run `./bin/goal fix --inplace selfhost` a second time; confirm no "fixed" lines
  and no diff (idempotent fixed point).

## Task 3: Run the project gates
**Status**: completed
**Files**: (none)
**Depends on**: Task 2
**Spec coverage**: FR-4; AC-3, AC-4, AC-5
**Verify**: `task check` && `task build` && `task fixpoint` all green.
