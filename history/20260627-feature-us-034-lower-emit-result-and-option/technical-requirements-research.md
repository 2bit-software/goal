# Technical Requirements / Research — US-034

## Where the work lands

- `internal/backend/lower.go` — add the open-E Result / Option encoders (mirroring
  the known-good `internal/pass/result.go` and `internal/pass/option.go`, but
  reading the parsed AST instead of token scans).
- `internal/backend/emit.go` — wire dispatch: lower the function signature
  (`Result[T,error]` -> named `(__goal_ok T, __goal_err error)` returns;
  `Option[T]` -> `*T`), the `return Result.Ok/Err` and `return Option.Some/None`
  constructors (function-scoped), and statement-position `match` over Result/Option.

## Key AST shapes (already produced by the parser)

- `Result[int, error]` return type -> `*ast.IndexListExpr{X: Ident "Result",
  Indices: [int, error]}`. `Option[int]` -> `*ast.IndexExpr{X: Ident "Option",
  Index: int}`.
- `return Result.Ok(n)` -> `ReturnStmt{Results: [CallExpr{Fun: SelectorExpr{X:
  Ident "Result", Sel: "Ok"}, Args: [n]}]}`. `Option.None` is a bare
  `SelectorExpr`; `Option.Some(x)` is a `CallExpr`.
- A statement-position match -> `ExprStmt{X: *ast.MatchExpr}`. Arm patterns are
  `*ast.VariantPattern{Enum: Ident "Result"/"Option", Variant, Binding}`; arm
  `Body` is a generic `ast.Node` (Expr, Stmt, or BlockStmt).

## Encodings (mirror internal/pass, behavioral-tier exact text not required)

- Result open-E signature: `(__goal_ok T, __goal_err error)`.
  - `return Result.Ok(X)` -> `return X, nil`.
  - `return Result.Err(X)` -> `return __goal_ok, X`.
  - statement match: `lhs, __goal_err := scrut; if __goal_err != nil { errBody }
    else { okBody }`, where the Ok binding is renamed to `__goal_v` (lhs is `_`
    when the Ok binding is unused) and the Err binding is renamed to `__goal_err`.
- Option signature: `*T`.
  - `return Option.None` -> `return nil`.
  - `return Option.Some(x)` -> `return &x` for an addressable identifier, else box
    `__goal_some := x; return &__goal_some`.
  - statement match: `if __goal_o := scrut; __goal_o != nil { [binding := *__goal_o]
    someBody } else { noneBody }` (Some binding kept by name, declared only when used).

## Gensym names (reuse from internal/pass for now; US-035 retires the `__goal_`
prefix for `?`): `__goal_ok`, `__goal_err`, `__goal_v`, `__goal_some`, `__goal_o`.

## Constraints

- Function-scoped state: the `return` constructor lowering depends on the enclosing
  function's mode (open-E Result vs Option). Track current-function kind on the
  emitter and save/restore for nested func literals.
- Closed-E Result match (callee mode `ModeResultClosed`) must NOT be mis-lowered by
  the open-E path — guard it (defer to US-037). Use `sema.Info.FuncSignatures` to
  detect it.
- Scope the new behavioral-tier test to the 03-result and 04-option corpus cases.
- Zero-dependency; stdlib `testing` only.
