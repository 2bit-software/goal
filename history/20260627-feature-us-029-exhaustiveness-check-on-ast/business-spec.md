# Business Spec — US-029 Match exhaustiveness over the AST

## Outcome

A goal developer who writes a `match` over an enum gets the same exhaustiveness
guarantee from the new AST-based semantic analyzer that the legacy lexical checker
provides today:

- A `match` on a known enum that omits a variant and has no `_` rest-arm is
  **rejected** with an Error naming the missing variant(s) in declaration order.
- A `match` that covers every variant, or supplies an explicit `_` rest-arm, is
  **accepted** silently.
- A `match` whose arms name an enum not declared in this file cannot be proven
  complete, so the checker **defers** with a located Warning rather than guess.
- A `match` on the builtin sum types `Result`/`Option` is **not** this guarantee's
  concern and is skipped silently.

The diagnostic must land on the `match` keyword's line and be position-independent:
it fires regardless of match position (statement, `return match`, `var x = match`,
or the untyped `x := match`).

## Acceptance criteria

1. `internal/sema` implements the match exhaustiveness check over the AST.
2. Every exhaustiveness-related case in `testdata/check` (the `02-match` cases)
   passes through the sema checker via the corpus runner.
