# Implementation Plan — US-019 Parse expressions with Pratt and postfix ?

## File Inventory

### New Files
None. The change is contained within the existing parser package.

### Modified Files
| File | Changes |
|------|---------|
| `internal/parser/parser.go` | Replace the minimal `parseExpr` (operand+postfix only) with precedence-climbing binary parsing + unary/prefix parsing; add postfix `?`→`UnwrapExpr` to the postfix loop; add a `precedence(token.Kind) int` helper and a `parseUnary` method; broaden `startsExpr` to include unary-operator and `*`/`&` starts. Update package/section doc comments. |
| `internal/parser/parser_test.go` | Add `TestParseExpressionPrecedence` asserting `f(x)?`, `a.b?`, a mixed-precedence binary (`a + b * c == d`), left-assoc (`a - b - c`), and unary-tighter-than-binary (`-a * b`). |

## Package Structure

```
internal/parser/
  parser.go        (modified — expression tier upgraded)
  parser_test.go   (modified — new precedence/unwrap test)
```

No new packages; `internal/parser` keeps importing only `lexer`, `token`, `ast`.

## Dependency Graph

1. `precedence(kind)` helper + `parseUnary` (foundation, no new deps).
2. `parseBinary(minPrec)` precedence-climbing loop (uses 1).
3. `parseExpr` rewired to call `parseBinary(LowestBinaryPrec=1)` (uses 2).
4. Postfix `?` case added to `parsePostfix` → `ast.UnwrapExpr` (independent of 1-3).
5. `startsExpr` broadened so unary/`*`/`&` operands start expressions in
   statement/return position (uses nothing; supports callers).
6. Test (uses 1-5 through the public `ParseFile`).

## Interface Contracts

```go
// Lowest binary precedence (|| level). 0 means "not a binary operator".
func precedence(k token.Kind) int

// parseExpr parses a full expression: precedence-climbing binary over unary
// operands, each operand carrying its postfix chain (selector/call/index/?).
func (p *parser) parseExpr() ast.Expr            // rewired

// parseBinary parses a binary-operator expression with precedence >= minPrec,
// left associative.
func (p *parser) parseBinary(minPrec int) ast.Expr

// parseUnary parses a prefix-operator expression (+ - ! ^ & <-) or *x
// (StarExpr), else falls through to parsePostfix(parseOperand()).
func (p *parser) parseUnary() ast.Expr
```

Postfix `?` in `parsePostfix`:
```go
case token.QUESTION:
    q := p.advance()
    x = &ast.UnwrapExpr{X: x, Question: q.Pos}
```

Precedence table (Go semantics):
```
||                       -> 1
&&                       -> 2
== != < <= > >=          -> 3
+ - | ^                  -> 4
* / % << >> & &^         -> 5
(anything else)          -> 0
```

Unary/prefix starts: `ADD SUB NOT XOR AND ARROW` → `*ast.UnaryExpr`;
`MUL` → `*ast.StarExpr`.

## Integration Points

- `parseExpr` is the single entry point used by `parseExprList`,
  `parseValueSpec`, `parseSimpleStmt`, `parseReturnStmt`, control-clause headers,
  array length, index suffix, and composite elements — all gain full precedence
  automatically.
- `exprLev` brace-suppression is preserved: `parseUnary`/`parseBinary` do not
  touch `exprLev`; the existing `parsePostfix` `LBRACE` guard still governs
  composite literals in headers.
- `?` is added inside the existing `parsePostfix` loop, so it binds tighter than
  binary/unary and composes with the selector/call/index chain.

## Testing Strategy

- Add `TestParseExpressionPrecedence` to `internal/parser/parser_test.go`
  (package `parser`, stdlib `testing` only — no testify). Parse small expression
  snippets by wrapping them in a minimal function body (e.g.
  `package p\nfunc f() { _ = <expr> }`) or reuse the existing test's parsing
  helper, then walk the resulting `ast.AssignStmt`/`ast.ExprStmt` to assert node
  types and nesting:
  - `f(x)?` → `*ast.UnwrapExpr` wrapping `*ast.CallExpr`.
  - `a.b?` → `*ast.UnwrapExpr` wrapping `*ast.SelectorExpr`.
  - `a + b * c == d` → `*ast.BinaryExpr{Op: EQL}` with left
    `*ast.BinaryExpr{Op: ADD}` whose right is `*ast.BinaryExpr{Op: MUL}`.
  - `a - b - c` → left-nested `*ast.BinaryExpr{Op: SUB}`.
  - `-a * b` → `*ast.BinaryExpr{Op: MUL}` with left `*ast.UnaryExpr{Op: SUB}`.
- Keep all existing parser tests untouched and green.
- Project gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
