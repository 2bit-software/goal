# Implementation Report — US-022 Gate interp on native sema only

## What was implemented

The goscript interpreter's `Run()` now gates on native sema checks before any
evaluation. A program that violates a static guarantee (e.g. a non-exhaustive
`match`) is refused with a located, named error BEFORE `func main` is found or
run; warnings (located deferrals) do not block.

## Changes

- `internal/interp/interp.go`:
  - `Run()` calls the new `gate()` first.
  - `gate()` iterates `sema.Check(ip.file, ip.info)` and returns
    `interp: refused before run: <line:col>: [<code>] <message>` for the first
    `sema.Error`, skipping `sema.Warning`; nil otherwise. Uses only the
    already-imported `goal/internal/sema` — no new dependency.
- `internal/interp/gate_test.go` (new):
  - `TestRunRefusesNonExhaustiveMatch` — refusal names the code and is located.
  - `TestRunAllowsExhaustiveMatch` — clean program runs (no false refusal).
  - `TestRunDoesNotBlockOnWarning` — unresolved-enum warning does not refuse.
  - `TestInterpHasNoGoTypesOrTypecheckDep` — `go list -deps goal/internal/interp`
    contains neither `go/types` nor `goal/internal/typecheck`.

## Verification

- `go build ./...` — OK
- `go vet ./...` — OK
- `go test ./... -count=1` — all packages OK
- `go test ./internal/interp -run 'TestRun|TestInterpHasNo'` — OK

## Notes / gotcha

- The parser does NOT insert a semicolon after a bare `return` before a
  newline, so a `=> return` match arm followed by another arm swallows the next
  arm's pattern as the return expression. Use a block-body arm (`=> {}`) or a
  `return <value>` when a match arm must precede another arm in a test fixture.
