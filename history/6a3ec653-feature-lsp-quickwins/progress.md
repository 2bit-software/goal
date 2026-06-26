# Progress Log

All tasks complete. `go build ./...`, `go vet ./internal/lsp/...`, `gofmt -l`, and full
`go test ./...` pass. Binary rebuilt + installed; capabilities verified over stdio.

### T001 — Protocol types + capabilities — Complete
- `internal/lsp/protocol.go`: CodeAction*, WorkspaceEdit/TextDocumentEdit/TextEdit,
  DocumentSymbol*, CodeActionOptions, SymbolKind consts; extended ServerCapabilities.
- `server.go` initialize advertises `codeActionProvider` (source.fixAll, source.fixAll.goal)
  and `documentSymbolProvider: true`.

### T002 — Precise diagnostic ranges — Complete
- `diagnostics.go`: `toLSP(text, tokEnd, d)` uses the token's end offset; `tokenEnds` builds
  the start→end map per file; both call sites updated; fallback to line-end when the offset
  isn't a token start or the end ≤ start.

### T003 — Document symbols — Complete
- `internal/lsp/symbols.go`: `documentSymbols` handler + `collectSymbols` two-phase walk
  (`scanDecls` + `declEnd`). Ranges bounded by the next decl's keyword; alias detected via
  `=` and its line skipped; method = `func` followed by `(`; from/derive via preceding kw.
  Best-effort, never panics.

### T004 — Code action — Complete
- `internal/lsp/codeaction.go`: `codeActions` + `wantsKind`; `buffer` helper on Server.
  Version-pinned full-document WorkspaceEdit from `fix.File` when non-no-op; `context.only`
  honored; unknown URI / no-op → `[]`.

### T005 — Request handlers — Complete
- `server.go` `handle()`: `textDocument/codeAction` + `textDocument/documentSymbol` reply via
  `s.reply`; handlers read `s.docs` under `s.mu`, never touch `analysisMu`.

### T006 — Tests — Complete
- `codeaction_test.go` (offers/no-op/only-filter/unknown/broken-no-panic),
  `symbols_test.go` (kinds, bodyless-not-swallowing regression, empty/partial, handler),
  range test in `diagnostics_test.go`, capabilities assertions in `server_test.go`.
  11 new tests; existing tests migrated to the new `toLSP` signature.

### T007 — Verify + deploy — Complete
- Full suite green; `task install` rebuilt `goal` → GOBIN (12:00); `initialize` over stdio
  confirms both providers. No `editors/vscode/package.json` change needed.

## Decisions / Notes
- Idiomatize delivered as a `source.fixAll.goal` CodeAction with a version-pinned edit (no
  executeCommand round-trip); works on-save and via manual Quick Fix.
- OQ-1 (unfixable `fix` reports as info diagnostics) and OQ-2 (enum-variant/struct-field child
  symbols) deferred to follow-ups.
- **User action required**: reload the VSCode window to pick up the new server.
