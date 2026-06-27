# Verify — Acceptance — US-022

## Result: PASS (no CRITICAL/MAJOR)

Acceptance criteria from the story / spec, each mapped to evidence:

1. "The interpreter run path validates input solely through internal/sema
   (native AST checks) and refuses a program that fails a sema guarantee
   before evaluation."
   -> `Run()` calls `gate()` first; `gate()` uses only `sema.Check`.
   Evidence: `TestRunRefusesNonExhaustiveMatch` (refusal before eval),
   `TestRunAllowsExhaustiveMatch` (no false refusal). PASS.

2. "A unit test asserts a non-exhaustive-match program is refused with a
   located diagnostic."
   -> `TestRunRefusesNonExhaustiveMatch` asserts the error contains the code
   `non-exhaustive-match`, the `refused before run` prefix, and a `line:col`
   location. PASS.

3. "...a dependency test (via `go list -deps` or a source scan) asserts
   internal/interp does not depend on internal/typecheck or go/types."
   -> `TestInterpHasNoGoTypesOrTypecheckDep` runs
   `go list -deps goal/internal/interp` and fails on `go/types` or
   `goal/internal/typecheck`. PASS.

## Full gate verification (prd.json verifyCommands)

- `go build ./...` — OK
- `go vet ./...` — OK
- `go test ./... -count=1` — all packages OK

## Assumptions

- Surfacing the FIRST Error-severity diagnostic is sufficient to refuse (the
  spec marks "render all diagnostics" out of scope).
- Warnings (located deferrals) intentionally do not block, matching FR-3.
