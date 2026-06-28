# Technical Requirements / Research — US-014

## Existing seams (from US-013 statement-position match)

- `internal/interp/interp.go` already has `execMatch` (statement position),
  intercepted in `execStmt`'s `*ast.ExprStmt` case via `s.X.(*ast.MatchExpr)`.
- Arm selection: walk `m.Arms`, match `*ast.VariantPattern` whose
  `p.Variant.Name == subj.Variant.Tag`, else fall to a `*ast.RestPattern` arm,
  else a loud `panicSignal` ("unreachable: non-exhaustive match ...").
- Payload binding: `execArm` opens `scope.NewChild()` and binds
  `vp.Binding.Name` to the whole variant value.

## Approach

- Value-position match flows through `evalExpr` (RHS of `:=`/`var =`, and the
  result of `return`), so add a `case *ast.MatchExpr` to `evalExpr` ->
  `evalMatch`.
- `evalMatch` mirrors `execMatch` but evaluates the selected arm body as an
  expression and returns its `Value`.
- Factor the shared arm-selection and arm-scope logic so statement- and
  value-position match stay in lock-step (one place to change dispatch).
- A value-position arm body is an `ast.Expr` (parser: `=> expr`); evaluate it
  via `evalExpr`. A non-expression body in value position is a descriptive
  refusal.
- Statement-position interception in `execStmt` is unchanged, so a bare
  `match {}` statement still routes to `execMatch` — no conflict.

## Out of scope

Result/Option as tagged unions (US-015/016), `?` unwinding (US-017).
