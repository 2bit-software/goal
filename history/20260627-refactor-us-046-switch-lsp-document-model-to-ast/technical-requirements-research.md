# Technical Requirements / Research — US-046

## Current state

- `internal/lsp/symbols.go` builds the document outline via `scan.Lex` +
  `scanDecls` (a hand-rolled token walk tracking bracket depth and keyword
  positions) — this is the token-scan machinery the rewrite is retiring.
- `internal/lsp/diagnostics.go` uses `check.Analyze` /
  `check.AnalyzePackageInDirWith` for findings and `scan.Lex` (via
  `tokenEnds`) to widen a diagnostic to its token span.

## Direction (per REWRITE-ARCHITECTURE.md and progress.txt patterns)

- Derive document symbols from `parser.ParseFile` over `internal/ast`:
  walk `ast.File.Decls` and map each declaration (EnumDecl, SealedInterfaceDecl,
  TypeSpec struct/interface/alias, FuncDecl incl. method + from/derive modifiers)
  to a `DocumentSymbol` with Range/SelectionRange from node Pos()/End() offsets.
- Best-effort: a declaration the parser cannot read is skipped, never fatal.
  When the source does not parse at all, fall back gracefully (empty outline)
  so the LSP never errors.
- Ranges must match existing symbols_test.go expectations — verify byte offsets
  from node positions reproduce the prior keyword-start..body-end spans.
- Keep `check.OffsetToPosition` for offset->Position conversion.

## Tests

- `internal/lsp/symbols_test.go` is the behavioral contract for the outline.
- Full `internal/lsp` test suite must stay green.
