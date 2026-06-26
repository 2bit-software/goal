---
status: complete
updated: 2026-06-25
---

# Research: Statement-context error passthrough in `goal fix`

## Executive Summary

`goal fix` collapses the value-binding error-propagation form (`x, err := f(); if err != nil …`)
but has no rule for the statement-context form (`if err := f(); err != nil …`), so the latter is
silently left untouched. The fix is a new fixer, `fixPropagateInit`, that detects the if-init
guard and reuses the existing `validPropagationReturn` body check to rewrite a provably-equivalent
passthrough to `f()?`. It runs alongside `fixPropagate` and matches a disjoint set of `if` shapes.

## Findings

### Codebase Context

`fix.File(src)` runs every fixer to a fixed point (`maxIters` loop). Each iteration re-lexes,
rebuilds `analyze.FuncSpans`, runs the fixers, and splices their `scan.Replacement`s. Fixers:

- `fixPropagate` — collapses the **value-binding** form: a binding on the line *above* an
  `if err != nil { … }`, inside a `ModeResult` / `ModeOption` function, to `value := rhs?`.
  Keys off `toks[i+5].Text == "{"`, i.e. `if err != nil {` directly — so an init clause makes
  `toks[i+5]` not the brace, and the candidate is silently skipped. **This is the gap.**
- `fixResultSig` — converts a plain Go `(T, error)` signature to `Result[T, error]`.
- `fixSwitchToMatch` — reports (does not rewrite) switch-over-enum candidates.
- `reportCallSites` — once converged, flags manual error guards inside non-Result functions.

Reused helpers: `analyze.SigAt`, `validPropagationReturn` (already accepts `return Result.Err(err)`
and `return zero, err`), `scan.SplitAssign`, `scan.MatchBrace`, `scan.IsIdent`, `scan.IsLineStart`,
`lineStartBefore`, `indentOf`, `spanHasComment`.

New helpers: `ifBodyBrace` (mirror of existing `switchBodyBrace`) and `topLevelSemicolon`.

### Domain Knowledge

- `text/scanner` lexes operators char-by-char: `!=` → `!` `=`, `:=` → `:` `=`, `;` is its own
  token. Existing code already relies on the `!`/`=` split; `SplitAssign` cuts the `:=` string in
  raw source, so no token-level `:=` handling is needed.
- The fixer's invariant: never emit behavior-changing code. Error-wrapping / decorating / non-zero
  bodies must be left untouched — `validPropagationReturn` already enforces this.

## Decision Points

- [x] **D1**: Scope to `ModeResult` only (open-E `Result[T, error]`) — matches `fixPropagate`'s
  `isResult` gate. Closed-E and Option are out of scope.
- [x] **D2**: Require single-variable init LHS equal to the condition variable. A value+error LHS
  is excluded because a bare `CALL?` statement discards the unwrapped value.
- [x] **D3**: New fixer rather than extending `fixPropagate` — keeps each function focused and the
  two match disjoint `if` shapes, so no replacement overlap.

## Recommendations

1. Add `fixPropagateInit` to `internal/fix/propagate.go`, invoked in `File` immediately after
   `fixPropagate`. Add `ifBodyBrace` and `topLevelSemicolon` helpers (the latter generalizes the
   existing depth-walk pattern used by `switchBodyBrace` / `topLevelComma`).
2. Add unit tests to `fix_test.go` covering the happy path, idempotence, wrapped/decorated body,
   comment Skip, `else`, and non-Result enclosing function.

## Sources

- `internal/fix/propagate.go`, `internal/fix/fix.go`, `internal/fix/match.go`,
  `internal/fix/callsite.go`, `internal/fix/resultsig.go`, `internal/fix/fix_test.go`
- `internal/scan/scan.go`, `internal/analyze/spans.go`, `internal/analyze/analyze.go`
