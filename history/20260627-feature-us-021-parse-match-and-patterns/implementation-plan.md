# Implementation Plan — US-021 Parse match and patterns

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/parser/goal_match.go` | `match` expression + arm + pattern parsing methods. |
| `internal/parser/goal_match_test.go` | Statement- and value-position match parse tests + rest/binding assertions. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/parser/parser.go` | Dispatch `token.MATCH` in `parseStmt` (→ `ExprStmt{MatchExpr}`) and `parseOperand` (→ `MatchExpr`); add `token.MATCH` to `startsExpr`. |

## Package Structure

```
internal/parser/
  parser.go          (modified: 3 small hooks)
  goal_decl.go       (unchanged)
  goal_match.go      (new: parseMatchExpr/parseMatchArm/parsePattern/parseVariantPattern)
  goal_match_test.go (new)
```

## Dependency Graph

1. AST nodes (`ast.MatchExpr/MatchArm/VariantPattern/RestPattern`) — already exist.
2. `goal_match.go` parsing methods — depend on (1) and the existing `parseExpr`/`parseBlock`/`exprLev` machinery.
3. `parser.go` dispatch hooks — depend on (2).
4. Tests — depend on (3).

## Interface Contracts

```go
// goal_match.go
func (p *parser) parseMatchExpr() *ast.MatchExpr
func (p *parser) parseMatchArm() *ast.MatchArm
func (p *parser) parsePattern() ast.Expr        // RestPattern or VariantPattern
func (p *parser) parseVariantPattern() ast.Expr // Enum.Variant(binding)?
```

`parseMatchExpr`:
- consume `match`; parse subject with `exprLev = -1` (suppress composite braces);
  restore `exprLev`; expect `{`; loop arms until `}`; expect `}`.

`parseMatchArm`:
- `parsePattern()`; expect `FAT_ARROW`; body = `parseBlock()` if `{`, else `parseExpr()`.

`parsePattern`:
- `_` (IDENT with Lit "_") → `*ast.RestPattern`; else `parseVariantPattern()`.

`parseVariantPattern`:
- read dotted name; last segment = Variant (`*Ident`), prefix = Enum (`*Ident`
  or `*SelectorExpr`, nil if no dot); optional `(IDENT)` → Binding.

## Integration Points

- `parser.go` `parseStmt` switch: `case token.MATCH: return &ast.ExprStmt{X: p.parseMatchExpr()}`.
- `parser.go` `parseOperand` switch: `case token.MATCH: return p.parseMatchExpr()`.
- `parser.go` `startsExpr`: add `token.MATCH` so `return match ...` parses results.

## Testing Strategy

- Parse `features/02-match/examples/status_match.goal` (statement position):
  assert `handle`'s body has a single `ExprStmt` wrapping a `*ast.MatchExpr` with
  3 arms; assert arm patterns are `*ast.VariantPattern` with the expected
  variant names and that `Status.Active(a)` records Binding "a".
- Parse `status_var.goal` and `status_return.goal` (value position): locate the
  `*ast.MatchExpr` in the var initializer / return result and assert 3 arms.
- Parse `status_rest.goal`: assert the last arm pattern is a `*ast.RestPattern`.
- Assert `ParseFile` returns no error for each.
- All tests are `package parser` (internal), stdlib `testing` only.
