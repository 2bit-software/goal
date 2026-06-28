# Tasks — US-005 Eval literals and operators

All tasks: completed.

## Task 1 (completed): Implement evalExpr in eval.go
- Create `internal/interp/eval.go` (`package interp`).
- `func (ip *Interp) evalExpr(expr ast.Expr, scope *Env) (Value, error)` dispatch:
  - `*ast.ParenExpr` -> evalExpr of X.
  - `*ast.BasicLit` -> decode by Kind: INT (strconv.ParseInt base 0) -> IntVal;
    FLOAT (ParseFloat) -> FloatVal; STRING (strconv.Unquote) -> StrVal;
    CHAR (strconv.Unquote -> rune) -> IntVal; else descriptive error.
  - `*ast.Ident` named `true`/`false` -> BoolVal; any other ident -> descriptive
    "undefined / not yet supported" error (identifiers are US-007).
  - `*ast.BinaryExpr` -> evalBinary.
  - `*ast.UnaryExpr` -> evalUnary.
  - default -> descriptive "unsupported expression %T" error.
- evalBinary: for `&&`/`||`, evaluate left, short-circuit; else evaluate both
  operands and apply arithmetic (+ - * / %), comparison (== != < <= > >=). `+`
  concatenates strings. Divide/remainder by zero -> named error. Kind mismatch
  -> descriptive error.
- evalUnary: `-` numeric negation, `!` boolean negation; else error.
- Verify: `go build ./internal/interp`.

## Task 2: Wire ExprStmt into execBlock
- In `internal/interp/interp.go`, change execBlock's loop to switch on statement
  type, adding `case *ast.ExprStmt:` calling `ip.evalExpr(s.X, scope)` and
  returning any error (value discarded). Leave other forms for later stories.
- Verify: `go build ./internal/interp && go vet ./internal/interp`.

## Task 3: Table-driven tests
- Create `internal/interp/eval_test.go` (`package interp`).
- `evalProgram(t, expr) Value` helper: parse `package main\nfunc main() {
  <expr> }` via parser.ParseFile, sema.Resolve, find the ExprStmt in main, call
  evalExpr; t.Fatalf on parse/eval error.
- `TestEvalExpressions`: >= 12 rows asserting result Value via Equal — int/float/
  string/bool literals, `+ - * / %` (int + float), string `+`, all comparisons,
  `&& ||` truth values, unary `- !`, parentheses/precedence.
- `TestShortCircuit`: `false && (1/0 == 0)` and `true || (1/0 == 0)` evaluate
  without error (right side never evaluated).
- `TestEvalErrors`: divide-by-zero and unsupported-operand-kind return errors.
- Verify: `go test ./internal/interp -count=1`.

## Task 4: Full verify gates
- Run `go build ./...`, `go vet ./...`, `go test ./... -count=1`. All green.
