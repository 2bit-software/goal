# Research Findings — US-046

## AST surface available

- `parser.ParseFile(src) (*ast.File, error)` is the entry point.
- `ast.File.Decls []Decl` holds top-level declarations. Relevant node types:
  - `*ast.EnumDecl` (Name, Enum kw pos) -> symEnum
  - `*ast.SealedInterfaceDecl` (Name, Sealed kw pos) -> symInterface (detail "sealed interface")
  - `*ast.GenDecl{Tok: token.TYPE}` with `[]*ast.TypeSpec`:
    - `TypeSpec.Assign != zero` (alias) -> symClass
    - `TypeSpec.Type` is `*ast.StructType` -> symStruct
    - `*ast.InterfaceType` -> symInterface
    - else -> symClass
  - `*ast.FuncDecl`: `Recv != nil` -> symMethod; else symFunction
    (from/derive modifiers stay symFunction; Pos() already points at the
    `from`/`derive` keyword via ModPos).
- Every node has `Pos()`/`End()` returning `token.Pos{Offset,Line,Col}`, so
  ranges come straight off node offsets — no token re-walk.

## Range/selection mapping (preserving symbols_test.go contract)

- Range = decl node Pos()..End() (for a single-spec TYPE GenDecl, start at the
  `type` keyword = GenDecl.Pos() to keep the keyword in range as before; for a
  grouped TYPE decl, use each spec's own Pos()).
- SelectionRange = the name Ident's Pos()..End().
- Node End() for bodyless from/derive/alias is on the same line as the decl, so
  the "bodyless must not swallow the next decl" test holds by construction.
- Broken/partial source: `parser.ParseFile` returns an error; `collectSymbols`
  returns an empty (non-nil) slice — no panic. Empty source likewise yields [].

## Diagnostics

- Findings still come from `check.Analyze` / `check.AnalyzePackageInDirWith`
  (the checker; migrating the checker itself to sema is separate corpus work,
  and the LSP package path depends on `analyze.EnrichForeign`). Out of scope.
- lsp's own token walk is `tokenEnds` (currently `scan.Lex`) used only to widen
  a diagnostic to its token span. Switch it to the AST front-end's lexer
  (`lexer.Tokens`): a token's end = `Pos.Offset + len(Lit)`. This removes the
  `internal/scan` import from internal/lsp entirely (the last scan use after
  symbols.go is rewritten). Diagnostics tests pass `tokEnd` explicitly and no
  server/package test asserts exact diagnostic ranges, so this is behavior-safe.

## Confidence: High — closed, test-pinned refactor over a known AST.
