# Verify — Quality — US-046

## Observations

- `collectSymbols` is now a thin parse + declaration walk; best-effort contract
  is explicit (parse error / nil File / nil name -> skip, never panic).
- Range semantics preserved: single-spec `type` decls keep the keyword in range
  (start = GenDecl.Pos()); grouped specs use their own Pos() to avoid overlap.
- `tokenEnds` token end = `Pos.Offset + len(Lit)`; the live publish path widens a
  diagnostic to its token, with the existing line-end fallback when an offset is
  not a token start (toLSP guards end<=start). No diagnostic-range test regressed.
- `internal/scan` fully removed from internal/lsp non-test code; the LSP document
  model and diagnostic range-widening now ride the AST front-end (parser + lexer).

## Edge cases

- Malformed `type Broken struct {` source: parser errors -> empty outline, no panic.
- Unknown URI / non-file URI: handler short-circuits to empty (`documentSymbols`).
- from/derive func bodyless: Pos() points at the modifier keyword; End() at the
  result type — range does not overrun the next decl.

## Findings

- None blocking. The checker findings still come from `check.Analyze` /
  `check.AnalyzePackageInDirWith` (intentionally out of scope — the LSP package
  path needs analyze foreign resolution).
