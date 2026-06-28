# Verify — Acceptance Coverage — US-046

Full suite green: `go build ./...`, `go vet ./...`, `go test ./... -count=1`
(all 21 packages ok). `internal/lsp` package green.

| Acceptance criterion | Evidence |
|----------------------|----------|
| Outline reports every decl form with correct kind | `TestCollectSymbolsKinds` (enum, struct, interface, sealed interface, alias, function, method, from/derive) — passes against the AST-derived `collectSymbols`. |
| Bodyless alias / from/derive does not swallow next decl | `TestCollectSymbolsBodylessDoesNotSwallow` — passes (node End() stops at the decl's own last token). |
| Selection range starts at/after full range start | `TestCollectSymbolsKinds` asserts `SelectionRange.Start.Line >= Range.Start.Line`. |
| Empty -> non-nil empty; malformed -> no panic | `TestCollectSymbolsEmptyAndPartial` — parse error returns `[]DocumentSymbol{}`. |
| Handler returns outline for open doc, empty for unknown | `TestDocumentSymbolHandler` (9 symbols / 0 for unknown URI). |
| `scanDecls` token walk removed | `grep -rn scanDecls internal/lsp` -> empty. `internal/scan` no longer imported by internal/lsp non-test files. |
| Diagnostics derived from AST front-end | `tokenEnds` now uses `lexer.Tokens` (the AST front-end lexer); diagnostics tests (`TestToLSPMapping`, `TestToLSPRangeUsesTokenEnd`) + server/package tests green. |
| Full LSP suite passes | `go test ./internal/lsp -count=1` ok. |

No acceptance criterion is uncovered.
