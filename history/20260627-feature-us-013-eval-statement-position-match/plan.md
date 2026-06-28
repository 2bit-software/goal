# Implementation Plan — US-013 Eval statement-position match

## Approach

Statement-position `match` parses to `*ast.ExprStmt{X: *ast.MatchExpr}`. Intercept
it in the interpreter's statement dispatch and dispatch on the universal
tagged-union variant tag (the same `Variant{TypeID,Tag,Fields}` enum construction
produces in US-012). Payload binding is the whole variant value; payload fields
read through it via the selector path.

## Changes

- `internal/interp/interp.go`
  - `execStmt` ExprStmt case: route a `*ast.MatchExpr` to a new `execMatch` BEFORE
    the existing CallExpr handling (value-position match stays out of scope).
  - `execMatch(m, scope)`: evaluate the scrutinee (must be `KindVariant`), find the
    arm whose `VariantPattern.Variant.Name == subj.Variant.Tag`, else a `_`
    `RestPattern` arm; with neither, raise a `panicSignal` carrying an `unreachable`
    message (defensive default of a proven-exhaustive match — loud, not silent).
  - `execArm`: bind the `VariantPattern.Binding` name to the whole variant value in
    a fresh child scope, then run the body.
  - `execArmBody`: dispatch the generic `ast.Node` arm body — `ast.Stmt` via
    execStmt (covers block / `return` arms), a `*ast.CallExpr` via evalCallMulti
    (void calls), any other `ast.Expr` via evalExpr.

- `internal/interp/eval.go`
  - `evalSelector`: read a payload field off a `KindVariant` receiver
    (`binding.field`) via `Value.Field`, in addition to the existing struct-field
    read; a non-variant/non-struct receiver stays a descriptive refusal.

## Tests (`internal/interp/match_test.go`)

A 02-match-shaped enum (`Point` data-less, `Circle{radius}`, `Square{side}`):
correct arm runs for each variant (incl. payload binding read), a `_` rest arm
runs when no variant matches, the defensive default panics `unreachable`
(hand-built match over an uncovered tag), and a non-variant subject is refused.

## Invariants

- No new dependency on go/types, internal/backend, or internal/typecheck (US-022).
- stdlib `testing` only (no testify).
