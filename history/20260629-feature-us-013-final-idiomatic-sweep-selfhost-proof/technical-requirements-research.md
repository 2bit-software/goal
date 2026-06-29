# Technical Requirements / Research — US-013

## Machine check for AC-1 (zero auto-convertible sites)

`goal fix` without `-inplace` always prints the (possibly unchanged) rewritten file to
stdout, so the reliable "did it convert anything" check is to run `goal fix -inplace`
on a COPY of the selfhost tree and `diff -r` against the original. Verified: across all
39 `selfhost/**/*.goal` files the diff is EMPTY, and the stderr report contains only
`skipped: [result-sig]` (refusals) and `suggestion: [call-site]` (advisory) lines — no
`fixed` lines. Neither a skip nor a suggestion is an auto-conversion, so AC-1 holds.

## Remaining deliberately-Go constructs

Every flagged function maps to a documented refusal in DECISIONS.md from the
per-package audits US-005..US-012 (token, lexer, ast, parser, sema, project, pipeline,
backend, typecheck), EXCEPT `selfhost/main.goal`'s `run` and `emitPackage`, which were
never given a per-package audit story. These are the whole-tree sweep's job (US-013):

- `run (error)` — the CLI entry point (called by `main`). Returns a bare `error` (no
  value channel). Its propagation either WRAPS usage errors via `fmt.Errorf` or is the
  top-level plumbing return; `goal fix` correctly `skipped: [result-sig]` ("returns a
  bare `error`; not auto-converted to Result"). Converting a bare-error CLI entry to
  Result buys nothing and the result-sig rule refuses it.
- `emitPackage (error)` — IO helper called only by `run`; propagates `os.MkdirAll` /
  `os.WriteFile` errors. Same bare-error refusal class.

Both are recorded in a new DECISIONS.md "US-013" section.

## Verification commands (from prd.json verifyCommands)

- `task check` — go vet + full `go test ./...` (includes internal/corpus transpile +
  behavioral + check tiers and the internal/selfhost port behavioral gates that build
  the goal-emitted packages and run their tests).
- `task build` — builds both binaries.
- `task fixpoint` — bootstraps stage-0 -> goal-c-1 -> goal-c-2 and diffs goal-c-1 vs
  goal-c-2 emit over ./selfhost (byte-identical proof).
