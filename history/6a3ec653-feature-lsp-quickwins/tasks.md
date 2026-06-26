# Tasks: High-ROI LSP quick wins

Complexity: **Medium** (~9 files). T001 and T002 are independent; T003/T004 depend on T001;
T005 depends on T003+T004; T006 on all; T007 deploys. Tasks trace to plan steps and spec FRs.

- [ ] **T001 [P]** `internal/lsp/protocol.go`: add `CodeActionParams`, `CodeActionContext`,
  `CodeAction`, `WorkspaceEdit{DocumentChanges}`, `TextDocumentEdit`, `TextEdit`,
  `DocumentSymbolParams`, `DocumentSymbol`, `CodeActionOptions`, the `SymbolKind` const block
  (Class=5,Method=6,Field=8,Enum=10,Interface=11,Function=12,EnumMember=22,Struct=23); extend
  `ServerCapabilities` with `CodeActionProvider`/`DocumentSymbolProvider`. In `server.go`
  initialize result, advertise both (`codeActionKinds: ["source.fixAll","source.fixAll.goal"]`,
  `documentSymbolProvider: true`). Plan Step 1 / FR-002, FR-005.
  - **Accept**: builds; initialize response includes both providers.

- [ ] **T002 [P]** `internal/lsp/diagnostics.go`: change `toLSP` to
  `toLSP(text string, tokEnd map[int]int, d check.Diagnostic)`; build `tokEnd` (tok.Start→
  tok.End from `scan.Lex`) once per file and pass it at both call sites (`compile` per
  `views[i].src`, `compileSingle` from `text`). Use token end for the range; fall back to
  `lineLength` when `d.Pos ∉ tokEnd` or resolved end ≤ start. Plan Step 2 / FR-004.
  - **Accept**: range covers the token; multi-line end correct; fallback intact; existing
    diagnostics tests pass.

- [ ] **T003 [US3]** `internal/lsp/symbols.go` (new): `documentSymbols(src string)
  []DocumentSymbol` — two-phase depth-0 token walk per technical-spec §4. Phase 1 records
  top-level decls (enum/struct/interface/sealed/alias/func/method/from-func/derive-func) with
  kind + name token; Phase 2 bounds each decl's range by the *next* decl's keyword (body brace
  = first `{` before that bound → `MatchBrace`.End; else bodyless → `NextNewline`). Alias-ness
  from the `=` token; method iff `func` followed by `(`; `i>0` guard. SelectionRange = name
  span. Best-effort, never panics. Plan Step 3 / FR-005, FR-006.
  - **Accept**: each top-level decl returned once with correct kind/range/selectionRange; a
    bodyless `derive func`/`type X = …` followed by another decl does NOT swallow it; empty/
    partial source → `[]` / best-effort, no panic.
  - **Depends on**: T001.

- [ ] **T004 [US1]** `internal/lsp/codeaction.go` (new): `codeActions(raw) []CodeAction` +
  `wantsKind(only, kind)` + a `buffer(uri)` helper on Server. Honor `context.only`
  (empty or prefix of `source`/`source.fixAll`/`source.fixAll.goal`); look up buffer; run
  `fix.File`; when `out != text` return one `source.fixAll.goal` action titled
  "Idiomatize file (goal fix)" with a version-pinned full-document `WorkspaceEdit` (range
  `[0,0]`..`OffsetToPosition(text,len(text))` 0-based; NewText = `out`; Version = buffer
  version); else `[]`. Unknown URI → `[]`. Plan Step 4 / FR-001, FR-001a, FR-001b, FR-007, FR-009.
  - **Accept**: fixable → correct version-pinned edit; no-op → `[]`; `only` filter honored;
    unknown URI → `[]`; broken source → no panic.
  - **Depends on**: T001.

- [ ] **T005** `internal/lsp/server.go` `handle()`: add `textDocument/codeAction` and
  `textDocument/documentSymbol` cases that reply via `s.reply(m.ID, …)` with non-nil slices
  (`[]` not `null`). Handlers read `s.docs` under `s.mu`, never touch `analysisMu`. Plan
  Step 5 / FR-007, FR-008, FR-009.
  - **Accept**: framed request → correct JSON-RPC response; no deadlock; diagnostics unaffected.
  - **Depends on**: T003, T004.

- [ ] **T006** Tests (stdlib `testing`, reuse `NewServerWithIO`/`debounce<=0`/framed capture):
  `codeaction_test.go` (fixable/no-op/only-filter/version/unknown-URI/broken-source),
  `symbols_test.go` (each kind + ranges; bodyless-not-swallowing regression; empty; partial),
  range cases in `diagnostics_test.go` (token end, fallback, multi-line), capabilities
  assertion in `server_test.go`. Cover the spec FR→test table. Plan Step 6.
  - **Depends on**: T001–T005.

- [ ] **T007** Verify + deploy: `go build ./...`, `go vet ./...`, `gofmt -l`,
  `go test ./... -count=1`. Then `task install` (rebuild `goal`→GOBIN); tell the user to
  reload the VSCode window; confirm `editors/vscode/package.json` needs no change. Plan Step 7 / FR-010.
  - **Depends on**: T001–T006.

## Traceability (FR → task)
FR-001/001a/001b→T004,T006 · FR-002→T001,T006 · FR-003→T004,T006 · FR-004→T002,T006 ·
FR-005→T001,T003,T006 · FR-006→T003,T006 · FR-007→T004,T005,T006 · FR-008→T005,T006 ·
FR-009→T004,T005 · FR-010→T007. No orphan tasks.
