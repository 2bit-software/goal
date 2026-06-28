# Research — US-014 Eval value-position match

This is an internal interpreter feature; the design is fully determined by the
existing US-013 (statement-position match) implementation in this repo, not by
external sources.

## Findings (from the codebase)

- `internal/interp/eval.go` `evalExpr` is the single expression-evaluation entry
  point. All value positions for this story route through it:
  - `return match` -> `execReturn` evaluates the non-call result via `evalExpr`.
  - `x := match` -> `execAssign` evaluates each RHS via `evalExpr`.
  - `var x = match` -> `execDecl` evaluates each spec value via `evalExpr`.
  So a single `case *ast.MatchExpr` in `evalExpr` covers all three.

- `internal/interp/interp.go` `execMatch` already implements arm dispatch for
  statement position: variant-tag match, `_` rest fallback, and a loud
  `panicSignal` "unreachable" default. The value form must reuse this exact
  selection + payload-binding logic to stay in lock-step (the progress log's
  guidance: keep match tag-keyed and uniform).

- Statement-position match is intercepted earlier, in `execStmt`'s `*ast.ExprStmt`
  case, BEFORE `evalExpr` is reached — so adding a `*ast.MatchExpr` case to
  `evalExpr` does not disturb the statement path. No conflict.

- Arm bodies are a generic `ast.Node` (per the progress log): `=> expr` parses to
  an `ast.Expr` (the value-position arm), `=> return x` / `{ ... }` to statements.
  Value-position evaluation reads the arm body as an `ast.Expr`.

## Decision

Add `evalMatch` to eval.go, factor shared `selectMatchArm` + `armScopeFor` helpers
out of `execMatch`/`execArm` so statement- and value-position match share one
dispatch. Confidence: High (mirrors shipped US-013 code).
