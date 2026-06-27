# Implementation Tasks — US-023

## Task 1: Add AST nodes (AssertStmt, DocComment, Doctest) + FuncDecl.Doc + Walk
**Status**: completed
**Files**: `internal/ast/goal_stmt.go` (new), `internal/ast/ast.go` (mod),
`internal/ast/walk.go` (mod)
**Depends on**: none
**Spec coverage**: FR-1 (FuncDecl.Doc unaffected; Mod already exists), FR-2
(AssertStmt), FR-3 (DocComment/Doctest), Walk acceptance criterion.
**Verify**: `go build ./internal/ast/...`

### Instructions
- New file `internal/ast/goal_stmt.go`:
  - `AssertStmt struct { Assert token.Pos; Cond Expr; Comma token.Pos; Msg Expr;
    Args []Expr }` with `Pos()` = Assert, `End()` = end of last non-nil of
    Args/Msg/Cond, and `func (*AssertStmt) stmtNode() {}` so it is a Stmt.
  - `DocComment struct { Slash token.Pos; Lines []string; Doctests []*Doctest }`
    with `Pos()`/`End()` (End derived from Slash + length of last line; a simple
    Slash-based end is fine). It is a support Node (no category marker), like
    Field/Variant.
  - `Doctest struct { Input string; Expected []string }` — plain data, not a Node.
- `internal/ast/ast.go`: add `Doc *DocComment` as the FIRST field of `FuncDecl`
  (keyed literals everywhere, so safe). Update the doc comment on FuncDecl.
- `internal/ast/walk.go`: add a case for `*AssertStmt` walking Cond, then Msg,
  then each Arg (use walkExpr/walkExprList, nil-guarded). Add a case for
  `*DocComment` (no Node children → visit only). In the existing `*FuncDecl`
  case, descend into `n.Doc` (Walk(v, n.Doc) guarded by != nil) before/after the
  existing children.
- Follow the nil/zero-Pos guard style already in walk.go and goal_decl.go.

## Task 2: Parser support — assert, from/derive, doctest attachment
**Status**: completed
**Files**: `internal/parser/goal_stmt.go` (new), `internal/parser/parser.go` (mod)
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-2, FR-3.
**Verify**: `go build ./internal/parser/...`

### Instructions
- `internal/parser/parser.go::newParser`: change the trivia filter to skip ONLY
  `token.COMMENT` (keep `token.DOC_COMMENT` in the stream).
- `parseFile` loop: before `p.parseDecl()`, call `doc := p.collectDoc()`. After a
  non-nil decl, if it is a `*ast.FuncDecl` and `doc != nil`, set `fd.Doc = doc`.
  Guard: if after collecting a doc the cursor is at EOF, break.
- `parseBlock` loop: if `p.at(token.DOC_COMMENT)`, `p.advance()` and continue
  (defensive; the corpus has none inside bodies).
- `parseStmt`: add `case token.ASSERT: return p.parseAssertStmt()`.
- `parseDecl` default arm (beside the `sealed` branch): if
  `p.isContextual("from") && p.peekKind()==token.FUNC` consume the `from` IDENT,
  call `fd := p.parseFuncDecl()`, set `fd.Mod = ast.FuncFrom`, `fd.ModPos = <pos>`,
  return fd. Same for `derive`/`ast.FuncDerive`.
- New file `internal/parser/goal_stmt.go`:
  - `parseAssertStmt`: expect ASSERT; `Cond = p.parseExpr()`; if `p.at(COMMA)`
    consume it (record Comma pos), `Msg = p.parseExpr()`, then while `p.at(COMMA)`
    consume + append `p.parseExpr()` to Args. (parseExpr stops at a top-level
    comma, so call-internal commas stay in Cond.)
  - `collectDoc()`: if not at DOC_COMMENT return nil; else gather the consecutive
    DOC_COMMENT run into Lines (strip the leading `///` and at most one following
    space), set Slash to the first token's Pos, set Doctests =
    extractDoctests(Lines), return the *ast.DocComment.
  - `extractDoctests(lines []string) []*ast.Doctest`: scan lines; a line whose
    trimmed text starts with `>>>` opens a new Doctest (Input = text after `>>>`
    trimmed); subsequent non-`>>>` lines append to the current Doctest's Expected
    until the next `>>>` or end. Lines before the first `>>>` are prose (ignored
    for doctests but retained in DocComment.Lines).

## Task 3: Tests over the example inputs
**Status**: completed
**Files**: `internal/parser/goal_stmt_test.go` (new)
**Depends on**: Task 2
**Spec coverage**: all acceptance criteria.
**Verify**: `go test ./internal/parser/... -count=1`

### Instructions
- `package parser`, stdlib `testing` only. Read inputs from `../../features/...`
  (cwd = internal/parser), as goal_decl_test.go does. Helper to read+ParseFile,
  failing on a non-nil error.
- `TestParseAssert`: parse 10-assert bank/message/multiple; locate the AssertStmt
  in each function body; assert bare (Msg==nil, len(Args)==0) vs message
  (Msg is *ast.BasicLit STRING, len(Args)==1 for message.goal). For multiple.goal
  assert the message assert's Cond is a *ast.BinaryExpr whose LHS is a CallExpr
  (`clamp(lo,hi,n)`) — proving the split fired only on the top-level comma — and
  that the `%`-bearing assert is bare.
- `TestParseFromDerive`: parse 12-derive-convert from_storage/slice/to_storage;
  assert FuncDecl.Mod == ast.FuncFrom / ast.FuncDerive on the right decls; assert
  Body==nil for the two bodyless derive funcs (fromStorage, toIDs) and Body!=nil
  for bodied ones (parseUUID, uuidToString, toStorage).
- `TestParseDoctests`: parse 11-doctests add/multi/mixed; assert the FuncDecl.Doc
  is non-nil and len(Doc.Doctests): add=1, multi=2; mixed: half=0, double=1.
  Spot-check add's Doctest.Input contains `add(2, 3)` and Expected contains `5`.
- `TestWalkNewNodes`: build (or parse) a tree containing an AssertStmt with Msg +
  Args and a FuncDecl with a Doc; run ast.Walk with a counting Visitor and assert
  it does not panic and visits the AssertStmt's Cond/Msg/Args.

## Task 4: Verify gates
**Status**: completed
**Files**: (none — verification only)
**Depends on**: Task 3
**Spec coverage**: build/vet/test acceptance criterion.
**Verify**: `go build ./... && go vet ./... && go test ./... -count=1`

### Instructions
Run the prd verifyCommands. All must be green. Fix any fallout before completing.
