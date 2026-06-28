# Plan Buildability Audit — US-046

## Checks

- Dependency order valid: symbols.go and diagnostics.go are independent edits;
  no forward references.
- Interface contracts agree: collectSymbols/tokenEnds signatures preserved, so
  callers (documentSymbols, compile/compileSingle, tests) compile unchanged.
- File paths verified to exist: internal/lsp/symbols.go, internal/lsp/diagnostics.go.
- AST node types verified present: ast.EnumDecl, ast.SealedInterfaceDecl,
  ast.GenDecl{Tok:token.TYPE}, ast.TypeSpec{Assign, Type}, ast.StructType,
  ast.InterfaceType, ast.FuncDecl{Recv}. token.Token{Pos.Offset, Lit}.
  parser.ParseFile(src)(*ast.File,error) and lexer.Tokens(src)[]token.Token
  confirmed by grep.
- Integration points specific: rangeOf(src, start, end) + check.OffsetToPosition
  reused for offset->Position; no new conversion logic.

## Findings

- None CRITICAL/MAJOR. MINOR: token end via Pos.Offset+len(Lit) assumes Lit is
  the verbatim lexeme text; correct for identifiers/operators/literals which is
  all the diagnostic-widening path needs.

## Assumptions

- parser.ParseFile returns (possibly nil) *ast.File plus error on malformed
  input; implementation guards on err and on a nil File before walking Decls.
- lexer.Tokens includes an EOF/sentinel token harmlessly (its zero-length Lit
  maps a start to itself; toLSP ignores end<=start). Verified behavior in impl.
