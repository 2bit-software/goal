# Technical Requirements / Research — US-017

## Inputs already in place
- `internal/token`: Kind constants + Pos{Offset,Line,Col} + Token{Kind,Lit,Pos}.
- `internal/lexer`: `lexer.Tokens(src) []token.Token` (flat stream ending in EOF);
  emits COMMENT/DOC_COMMENT trivia, goal lexemes (QUESTION/FAT_ARROW/ELLIPSIS).
- `internal/ast`: File/GenDecl/FuncDecl/ImportSpec/ValueSpec/TypeSpec + the Go
  type-expression nodes (Ident/SelectorExpr/StarExpr/ArrayType/MapType/StructType/
  InterfaceType/FuncType/ChanType/Ellipsis/IndexExpr) and BlockStmt.

## Design
- Hand-written recursive descent (REWRITE-ARCHITECTURE §3 / §1.4). No go/parser.
- Tokenize once via `lexer.Tokens`; skip COMMENT/DOC_COMMENT trivia for now
  (comment attachment is the fmt story US-045).
- No semicolon insertion: top-level decls dispatch on the leading keyword
  (package/import/type/var/const/func); grouped specs and field/param lists run
  until their closing delimiter; value-position operand parsing stops naturally.
- Type expressions parsed fully (needed for declaration shape): qualified names,
  pointer, array/slice, map, struct (with field list), interface, func, chan,
  single index.
- Function bodies are skipped as a balanced-brace `*ast.BlockStmt` (Lbrace/Rbrace
  set, List nil) — statement parsing is US-018.
- Declaration initializer values use a minimal operand+postfix expression parser
  (literals, ident, qualified ident, paren, composite literal, call, index, selector)
  — full precedence/unary/postfix-? is US-019.
- `ParseFile(src) (*ast.File, error)` returns the first parse error (with position)
  or nil.

## Test
- `internal/parser/parser_test.go` (package parser): parse a representative
  Go-subset sample (package, grouped+single+named+blank imports, grouped+single
  const/var/type incl. a struct, and func with receiver/params/results/body) and
  assert the Decl list shape (count, kinds, spec kinds, names, positions).
