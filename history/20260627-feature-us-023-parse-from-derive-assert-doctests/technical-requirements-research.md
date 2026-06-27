# Technical Requirements & Research — US-023

## Existing groundwork

- `internal/ast` already defines `FuncDecl.Mod`/`ModPos` with `FuncMod`
  constants `FuncPlain`/`FuncFrom`/`FuncDerive` (US-015). The parser does not
  yet set them.
- `from`/`derive` are contextual keywords lexed as `IDENT` (US-013), decided
  positionally — mirror the existing `sealed`/`implements` handling in
  `internal/parser/goal_decl.go` (`p.isContextual(word)`).
- `assert` is a reserved keyword: `token.ASSERT` exists (US-011).
- `///` lexes as a single `token.DOC_COMMENT` token (US-013). The parser
  currently SKIPS `COMMENT`/`DOC_COMMENT` as trivia
  (`internal/parser/parser.go` ~line 68). Doctests must instead be captured as
  structured nodes.

## Work required

1. AST: add an `AssertStmt` node (Cond, optional Msg format + Args) and a
   doctest node (e.g. `DocComment`/`Doctest` carrying raw `///` text lines and
   parsed `>>>` example/expected pairs). Add `Walk` cases + a descent test.
2. Parser:
   - `assert` statement: dispatch in `parseStmt`; parse the condition
     expression, then if a top-level comma follows, parse the format string and
     argument expressions (reuse `parseExpr`, split on top-level comma only).
   - `from func` / `derive func`: in the top-level decl dispatch, recognize the
     contextual `from`/`derive` IDENT before `func`, set `FuncDecl.Mod`/`ModPos`,
     and allow a bodyless `derive func` (signature with no `{ }`).
   - `///` doctests: attach DOC_COMMENT runs preceding a declaration as a
     structured doc node rather than skipping them.
3. Tests: parse `features/10-assert`, `features/11-doctests`, and
   `features/12-derive-convert` example `.goal` inputs and assert the produced
   nodes (assert forms, from/derive modifiers + bodyless, doctest structure).

## Constraints

- Zero-dependency; stdlib `testing` only (no testify).
- `internal/parser` imports only `lexer`/`token`/`ast`; tests stay
  `package parser` (internal).
- Keep the no-semicolon structural delimiting discipline from US-017/US-018.
