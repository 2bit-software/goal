# Implementation Plan — US-005 Eval literals and operators

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/eval.go` | Expression evaluation: evalExpr dispatch over *ast.BasicLit / *ast.BinaryExpr / *ast.UnaryExpr / *ast.ParenExpr; literal decoding; arithmetic/comparison/logical/unary operator semantics. |
| `internal/interp/eval_test.go` | Table-driven test evaluating >= 12 expression programs and asserting each result Value; plus short-circuit and error-path tests. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/interp/interp.go` | execBlock evaluates a `*ast.ExprStmt` (discarding its value) via evalExpr, so the statement-dispatch seam is consistent. Returns the eval error if any. |

## Package Structure

```
internal/interp/
  value.go       (existing) — Value model
  env.go         (existing) — scope chain
  interp.go      (existing) — New/Run/execBlock  [MODIFIED]
  eval.go        (new)      — evalExpr + operator helpers
  eval_test.go   (new)      — table-driven expression tests
```

## Dependency Graph

1. internal/ast, internal/token, internal/interp Value/Env (all existing).
2. eval.go evalExpr (depends on 1).
3. interp.go execBlock ExprStmt dispatch (depends on 2).
4. eval_test.go (depends on 2, plus internal/parser to build expressions).

## Interface Contracts

```go
// eval.go
func (ip *Interp) evalExpr(expr ast.Expr, scope *Env) (Value, error)

// helpers (unexported)
func evalBasicLit(lit *ast.BasicLit) (Value, error)
func (ip *Interp) evalBinary(b *ast.BinaryExpr, scope *Env) (Value, error)
func (ip *Interp) evalUnary(u *ast.UnaryExpr, scope *Env) (Value, error)
```

- evalExpr returns a descriptive error (not a panic) for: unsupported node kind,
  divide/remainder by zero, operator/operand-kind mismatch.
- Logical `&&`/`||` evaluate the left operand, then short-circuit: `&&` only
  evaluates the right when left is true; `||` only when left is false.

## Integration Points

- `internal/interp/interp.go` execBlock: add a `case *ast.ExprStmt:` that calls
  `ip.evalExpr(s.X, scope)` and returns the error. Other statement forms remain
  deferred to later stories (US-006+).
- Tests parse via `internal/parser.ParseFile`, reach
  `file.Decls[i].(*ast.FuncDecl).Body.List[0].(*ast.ExprStmt).X`, and call
  `evalExpr` directly (internal `package interp` test).

## Testing Strategy

- `eval_test.go`, `package interp` (internal — can call unexported evalExpr).
- A `evalProgram(t, exprSrc string) Value` helper wraps `package main\nfunc
  main() { <expr> }`, parses, resolves sema, and evaluates the single ExprStmt.
- Table-driven `TestEvalExpressions` with >= 12 rows covering: int/float/string/
  bool literals; `+ - * / %` on ints and floats; string `+`; each comparison op;
  `&& ||` truth table; unary `- !`; parentheses/precedence.
- `TestShortCircuit` proves the right operand is not evaluated (guarded `1/0`).
- `TestEvalErrors` proves divide-by-zero and unsupported-operand errors.
