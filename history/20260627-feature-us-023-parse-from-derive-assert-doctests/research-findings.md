# Research Findings — US-023

Self-contained parser story; research is codebase-internal. Key facts gathered
from `internal/{token,lexer,ast,parser}`:

## Tokens / lexer
- `token.ASSERT` is a reserved keyword kind (US-011).
- `from`/`derive` lex as `IDENT` (contextual), decided positionally — mirror
  `p.isContextual("sealed")` in `internal/parser/goal_decl.go`.
- `///` lexes as ONE `token.DOC_COMMENT` whose `Lit` is the FULL line text
  including the `///` prefix, no trailing newline (lexer.scanDocComment).

## AST
- `FuncDecl` already has `Mod FuncMod`/`ModPos token.Pos` with `FuncPlain`/
  `FuncFrom`/`FuncDerive` (US-015). `Body *BlockStmt` is nil for bodyless funcs.
- No `AssertStmt` node yet — must add (Stmt). No doctest node yet — must add.
- Category markers: a Stmt needs `stmtNode()`; Walk has a type switch to extend.

## Parser
- `newParser` currently STRIPS both `COMMENT` and `DOC_COMMENT` (parser.go ~L68).
  To make doctests structured, keep `DOC_COMMENT` in the stream and collect a
  leading run before each declaration; still strip ordinary `COMMENT`.
- `parseDecl` default arm already does contextual dispatch (`sealed`). Add
  `from func` / `derive func`: when `isContextual("from"/"derive")` and
  `peekKind()==FUNC`, consume the modifier, call `parseFuncDecl`, set Mod/ModPos.
  `parseFuncDecl` already leaves Body nil when no `{` follows → bodyless works.
- `parseStmt` dispatches on leading token — add `case token.ASSERT`.
- `parseExpr` stops at a top-level comma (comma is not a binary op), so the
  assert condition naturally absorbs call-internal commas (`clamp(lo,hi,n)`) and
  the first top-level comma separates the printf message + args.

## Plan
1. AST: `AssertStmt{Assert, Cond, Comma, Msg, Args}` (Stmt) + `DocComment{Slash,
   Lines, Doctests}` (Node) + `Doctest{Input, Expected}` data; `FuncDecl.Doc
   *DocComment`. Walk cases for AssertStmt (Cond/Msg/Args) + FuncDecl.Doc descent.
2. Parser: keep DOC_COMMENT tokens; `collectDoc()` before each decl, attach to
   FuncDecl; `parseAssertStmt`; from/derive dispatch in parseDecl; defensively
   skip stray DOC_COMMENT inside blocks.
3. Test: parse features/10-assert, 11-doctests, 12-derive-convert inputs; assert
   AssertStmt forms, FuncMod + bodyless, DocComment/Doctest structure.

**Confidence**: High — all groundwork exists; this wires parser + 2 AST nodes.
