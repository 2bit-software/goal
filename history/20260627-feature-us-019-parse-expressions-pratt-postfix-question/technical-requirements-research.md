# Technical Requirements / Research — US-019

## Existing state (internal/parser/parser.go)

- `parseExpr` currently does `parsePostfix(parseOperand())` — operand plus a
  selector/call/index/composite-literal postfix chain only. No binary, no unary,
  no `?`.
- `parseExprList`, `parseValueSpec`, `parseSimpleStmt`, control headers, return,
  composite elements all call `parseExpr` — they get full precedence for free
  once `parseExpr` is upgraded.
- `exprLev` already suppresses composite-literal braces in control headers; keep
  that behavior.

## Approach

- Introduce precedence-climbing binary parsing. Go's binary precedence levels
  (highest→lowest):
  - 5: `* / % << >> & &^`
  - 4: `+ - | ^`
  - 3: `== != < <= > >=`
  - 2: `&&`
  - 1: `||`
- Add a `precedence(token.Kind) int` helper in the parser (returns 0 for
  non-binary). `<-` (ARROW) is excluded as a binary op (channel direction/recv
  is a unary/type concern in this subset).
- Add unary/prefix parsing for `+ - ! ^ * & <-` producing `ast.UnaryExpr`
  (`*` and `&` use UnaryExpr/StarExpr as appropriate — keep `&`/`!`/`+`/`-`/`^`
  as UnaryExpr; `*x` as StarExpr to match Go AST conventions, `<-ch` as
  UnaryExpr with Op ARROW).
- Postfix `?` becomes `ast.UnwrapExpr{X, Question}`, applied in the postfix
  loop (binds tightest, after selector/call/index), so `f(x)?` and `a.b?` wrap
  the fully-built postfix operand.
- Left associativity via the standard `parseBinary(minPrec)` loop.

## Nodes (already exist in internal/ast)

- `BinaryExpr{X, OpPos, Op, Y}`
- `UnaryExpr{OpPos, Op, X}`
- `StarExpr{Star, X}`
- `UnwrapExpr{X, Question}` (internal/ast/goal_expr.go)

## Tests

- Add to internal/parser/parser_test.go: a test asserting `f(x)?`, `a.b?`, and a
  mixed-precedence binary (e.g. `a + b * c == d`) parse to the expected tree.
- Keep all existing parser tests and the full `go test ./...` suite green.
