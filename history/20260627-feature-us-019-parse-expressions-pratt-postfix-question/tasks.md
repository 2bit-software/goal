# Tasks — US-019 Parse expressions with Pratt and postfix ?

## Task 1 — Precedence helper + unary parsing (foundation)
**Files**: `internal/parser/parser.go`
**Depends on**: none
- Add `precedence(k token.Kind) int` returning Go's binary precedence (1..5),
  0 for non-binary. `<-` (ARROW) returns 0 (unary-only).
- Add `parseUnary()`: if current kind is `ADD/SUB/NOT/XOR/AND/ARROW`, consume and
  return `*ast.UnaryExpr{OpPos, Op, X: parseUnary()}`; if `MUL`, return
  `*ast.StarExpr{Star, X: parseUnary()}`; else fall through to
  `parsePostfix(parseOperand())`.
**Spec coverage**: FR-2.

## Task 2 — Precedence-climbing binary loop
**Files**: `internal/parser/parser.go`
**Depends on**: Task 1
- Add `parseBinary(minPrec int) ast.Expr`: `x = parseUnary()`; loop while
  `precedence(cur) >= minPrec`: consume op, `y = parseBinary(opPrec+1)` (left
  assoc), `x = &ast.BinaryExpr{X, OpPos, Op, Y}`.
**Spec coverage**: FR-1.

## Task 3 — Rewire parseExpr; add postfix `?`; broaden startsExpr
**Files**: `internal/parser/parser.go`
**Depends on**: Tasks 1-2
- `parseExpr()` → `return p.parseBinary(1)`.
- Add `case token.QUESTION:` to `parsePostfix`’s loop →
  `x = &ast.UnwrapExpr{X: x, Question: q.Pos}` (advance the `?`).
- Broaden `startsExpr` to include `ADD SUB NOT XOR AND ARROW MUL` (unary starts).
- Refresh the "Minimal expressions" section doc comment to describe the
  precedence-climbing grammar and postfix `?`.
**Spec coverage**: FR-1, FR-3, FR-4, FR-5.

## Task 4 — Test
**Files**: `internal/parser/parser_test.go`
**Depends on**: Tasks 1-3
- Add `TestParseExpressionPrecedence` asserting tree shape for `f(x)?`, `a.b?`,
  `a + b * c == d`, `a - b - c`, `-a * b`, parsed via `ParseFile` wrapping each
  expr in `func f() { _ = <expr> }`.
**Spec coverage**: AC verification for all FRs.

## Status
- Task 1 — completed
- Task 2 — completed
- Task 3 — completed
- Task 4 — completed

## Coverage check
- FR-1 → Tasks 2,3,4. FR-2 → Tasks 1,4. FR-3 → Tasks 3,4. FR-4 → Tasks 3,4.
  FR-5 → Task 3 + full suite.
- Plan file inventory: `internal/parser/parser.go` (Tasks 1-3),
  `internal/parser/parser_test.go` (Task 4). All covered.
