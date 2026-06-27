# Implementation Plan — US-023

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/ast/goal_stmt.go` | New AST nodes: `AssertStmt` (Stmt), `DocComment` (Node), `Doctest` (data). |
| `internal/parser/goal_stmt.go` | Parser methods: `parseAssertStmt`, `collectDoc`/`extractDoctests`, from/derive dispatch helper. |
| `internal/parser/goal_stmt_test.go` | Parses the 10-assert, 11-doctests, 12-derive-convert example inputs and asserts node shapes. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/ast/ast.go` | Add `Doc *DocComment` field to `FuncDecl`. |
| `internal/ast/walk.go` | Add Walk cases for `AssertStmt` (Cond/Msg/Args) and `DocComment`; descend into `FuncDecl.Doc`. |
| `internal/parser/parser.go` | Stop stripping `DOC_COMMENT` in `newParser` (keep COMMENT stripped); in `parseFile` collect a leading doc run and attach to a FuncDecl; defensively skip stray `DOC_COMMENT` in `parseBlock`; add `case token.ASSERT` in `parseStmt`; add from/derive dispatch in `parseDecl`. |
| `internal/ast/ast_test.go` | Extend the Walk-descent coverage to the new nodes (or add to goal_stmt_test). |

## Package Structure

```
internal/
  ast/
    goal_stmt.go      (new)  AssertStmt, DocComment, Doctest
    ast.go            (mod)  FuncDecl.Doc
    walk.go           (mod)  Walk cases
  parser/
    goal_stmt.go      (new)  parseAssertStmt, collectDoc, extractDoctests, from/derive
    goal_stmt_test.go (new)  example-driven parse tests
    parser.go         (mod)  newParser, parseFile, parseBlock, parseStmt, parseDecl
```

## Dependency Graph

1. AST nodes (`internal/ast/goal_stmt.go`, `FuncDecl.Doc`) — foundation.
2. Walk cases (`internal/ast/walk.go`) — depends on 1.
3. Parser methods + dispatch (`internal/parser/*.go`) — depends on 1.
4. Tests (`internal/parser/goal_stmt_test.go`, ast walk test) — depends on 1–3.

## Interface Contracts

```go
// internal/ast/goal_stmt.go
type AssertStmt struct {
    Assert token.Pos // position of "assert"
    Cond   Expr      // asserted condition
    Comma  token.Pos // position of "," before the message; zero if bare
    Msg    Expr      // printf format string (*BasicLit); nil for bare form
    Args   []Expr    // printf arguments; nil for bare form
}
func (s *AssertStmt) Pos() token.Pos { return s.Assert }
func (s *AssertStmt) End() token.Pos // end of last of Args/Msg/Cond
func (*AssertStmt) stmtNode() {}

// DocComment is a run of consecutive /// lines attached to a declaration.
type DocComment struct {
    Slash    token.Pos  // position of the first ///
    Lines    []string   // each /// line's text, /// prefix and one space stripped
    Doctests []*Doctest // extracted >>> examples
}
func (d *DocComment) Pos() token.Pos { return d.Slash }
func (d *DocComment) End() token.Pos
// (no category marker: a support Node like Field/Variant)

// Doctest is one >>> example: an input line and its expected-output lines.
type Doctest struct {
    Input    string   // text after ">>> "
    Expected []string // following non->>> lines
}

// internal/ast/ast.go
type FuncDecl struct {
    Doc *DocComment // attached /// doc run; nil if none
    // ...existing fields...
}
```

```go
// internal/parser/goal_stmt.go
func (p *parser) parseAssertStmt() *ast.AssertStmt
func (p *parser) collectDoc() *ast.DocComment        // consumes a leading DOC_COMMENT run
func extractDoctests(lines []string) []*ast.Doctest  // pure helper
```

## Integration Points

- `internal/parser/parser.go::newParser`: change the trivia filter to drop only
  `token.COMMENT`, keeping `token.DOC_COMMENT` in the stream.
- `parseFile` loop: `doc := p.collectDoc()` before `parseDecl`; if the decl is a
  `*ast.FuncDecl`, set `fd.Doc = doc`.
- `parseBlock` loop: if `p.at(token.DOC_COMMENT)` advance+continue (defensive;
  corpus has none inside bodies).
- `parseStmt`: `case token.ASSERT: return p.parseAssertStmt()`.
- `parseDecl` default arm (next to the `sealed` branch): if
  `p.isContextual("from")`/`("derive")` and `p.peekKind()==token.FUNC`, consume
  the modifier, call `p.parseFuncDecl()`, set `Mod`/`ModPos`.

## Testing Strategy

`internal/parser/goal_stmt_test.go` (package parser, stdlib testing only):
- `TestParseAssert`: parse bank/message/multiple `.goal`; assert AssertStmt
  count, bare vs message (Msg nil vs *BasicLit), arg counts, and that the
  message-split fired only on the top-level comma (Cond of the `clamp(...)` case
  is a BinaryExpr whose operands include the call).
- `TestParseFromDerive`: parse from_storage/slice/to_storage `.goal`; assert
  FuncDecl.Mod == FuncFrom/FuncDerive on the right decls and Body==nil for the
  two bodyless derive funcs, non-nil for bodied ones.
- `TestParseDoctests`: parse add/multi/mixed `.goal`; assert the attached
  DocComment and its Doctests count (add=1, multi=2, mixed: half=0, double=1).
- Walk coverage: a hand-built tree (or one of the parsed files) asserts Walk
  visits AssertStmt's Cond/Msg/Args and FuncDecl.Doc without panicking.
Example inputs are read from `../../features/...` (cwd = internal/parser), the
same convention as goal_decl_test.go / goal_match_test.go.
