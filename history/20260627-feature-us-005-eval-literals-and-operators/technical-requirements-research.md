# Technical Requirements / Research — US-005

## Existing seam

- internal/interp already has the Value model (value.go: IntVal/FloatVal/StrVal/
  BoolVal etc.), the Env scope chain (env.go), and the interpreter entry
  (interp.go: New/Run/execBlock). US-005 adds expression evaluation hanging off
  the per-statement dispatch seam.
- AST expression nodes (internal/ast/ast.go): *ast.BasicLit (Kind token.INT/
  FLOAT/STRING/CHAR; Value raw text), *ast.BinaryExpr (X, Op token.Kind, Y),
  *ast.UnaryExpr (Op, X), *ast.ParenExpr (X). Operator kinds live in
  internal/token (ADD/SUB/MUL/QUO/REM, EQL/NEQ/LSS/LEQ/GTR/GEQ, LAND/LOR, NOT).
- A bare expression in a function body parses to *ast.ExprStmt{X: <expr>}
  (parser.go parseSimpleStmt default case), so a test can parse
  `package main\nfunc main() { <expr> }`, reach into the FuncDecl body, and
  evaluate the ExprStmt's X.

## Approach

- Add internal/interp/eval.go with evalExpr(expr ast.Expr, scope *Env)
  (Value, error). Literal decoding via strconv (ParseInt base 0, ParseFloat,
  Unquote for strings/chars). Arithmetic/comparison promote to Go's int64/
  float64/string/bool with Go semantics. && / || short-circuit by evaluating the
  left operand first and only evaluating the right when needed.
- Wire execBlock's per-statement loop to evaluate *ast.ExprStmt (discarding the
  value) so the statement-dispatch seam stays consistent for later stories.
- Errors are located, named values (e.g. division by zero, unsupported operator)
  rather than panics, consistent with the interpreter's loud-refusal stance.

## Constraints

- Zero-dependency: stdlib + goal/internal/* only. Tests use stdlib testing (no
  testify).
- internal/interp must not import internal/backend or go/types (US-022 gate).
