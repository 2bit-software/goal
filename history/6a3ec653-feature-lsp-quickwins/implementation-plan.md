# Implementation Plan: High-ROI LSP quick wins

Ordered, dependency-aware. Each step ends compilable; tests are stdlib `testing` (no testify).
See `technical-spec.md` for code shapes, `research.md` for pinned decisions.

## Reuse posture

Reuse-first; the only genuinely new code is LSP glue + a token walk.

| Need | Reuse | New? |
|---|---|---|
| Idiomatization engine | `internal/fix.File` | reuse |
| Token lexing + brace/paren matching | `scan.Lex`, `scan.MatchBrace`, `scan.MatchParen`, `scan.FirstBodyBrace`, `scan.ParamsClose`, `scan.IsIdent` | reuse |
| Offset → line/col | `check.OffsetToPosition` | reuse |
| JSON-RPC request/response | `s.reply`, `rpcMessage` | reuse |
| Diagnostics range mapping | extend `toLSP` | extend |
| Code action handler | `codeaction.go` | **new** |
| Symbol token walk | `symbols.go` (`documentSymbols`) | **new** |
| Protocol types + capabilities | `protocol.go` additions | **new** |

`scan.ScanFuncs` is deliberately NOT reused for symbols (drops bodyless `from`/`derive func`,
no receiver/keyword info — research D5). `scan.Func` is left untouched (no blast radius into
`analyze`).

## Steps

### Step 1 — Protocol types + capability advertisement
- `protocol.go`: add CodeAction*, WorkspaceEdit/TextDocumentEdit/TextEdit, DocumentSymbol*,
  SymbolKind consts, extend `ServerCapabilities` (per technical-spec §1).
- `server.go`: initialize result advertises `codeActionProvider` (kinds) + `documentSymbolProvider`.
- **Accept**: builds; `initialize` response contains both providers (assert in test later).
- *Depends on*: none. *Risk*: low.

### Step 2 — Precise diagnostic ranges
- `diagnostics.go`: change `toLSP` to `toLSP(text, tokEnd map[int]int, d)`; build `tokEnd`
  per file (`scan.Lex`) in `compile` (per `views[i].src`) and `compileSingle`; fallback to
  `lineLength` when `d.Pos∉tokEnd` or end≤start.
- **Accept**: range covers the token; fallback intact; existing diagnostics tests pass.
- *Depends on*: none (independent of Step 1). *Risk*: low.

### Step 3 — Document symbols (`symbols.go`)
- New `documentSymbols(src string) []DocumentSymbol`: depth-0 token walk handling enum /
  struct / interface / sealed interface / type alias / func / method / from-func / derive-func
  with name, kind, range, selectionRange (technical-spec §4). Best-effort, never panics.
- **Accept**: returns each top-level decl once with correct kind/ranges; empty/partial source
  → `[]` / best-effort, no panic.
- *Depends on*: Step 1 (DocumentSymbol type + SymbolKind consts). *Risk*: medium (the walk).

### Step 4 — Code action (`codeaction.go`)
- New `codeActions(raw) []CodeAction`: `context.only` filter (`wantsKind`), buffer lookup,
  `fix.File`, version-pinned full-document `WorkspaceEdit` when `out != text`, else `[]`.
- Add `buffer(uri)` helper on Server (locks `s.mu`).
- **Accept**: fixable → fixAll action (edit NewText == `fix.File` out, version pinned); no-op
  → `[]`; `only` filter honored; unknown URI → `[]`; broken source → no panic.
- *Depends on*: Step 1 (CodeAction types). *Risk*: low–medium.

### Step 5 — Wire request handlers
- `server.go` `handle()`: add `textDocument/codeAction` and `textDocument/documentSymbol`
  cases replying via `s.reply`. Ensure handlers return non-nil slices (`[]` not `null`).
- **Accept**: framed request → correct JSON-RPC response; no deadlock with `analysisMu`.
- *Depends on*: Steps 3, 4. *Risk*: low.

### Step 6 — Tests
- `internal/lsp/codeaction_test.go`, `symbols_test.go`, range cases in `diagnostics_test.go`,
  capabilities assertion in `server_test.go`. Cover every FR per spec's FR→test table,
  including `only` filter, version pin, empty/unknown/partial, multi-line range.
- *Depends on*: Steps 1–5.

### Step 7 — Verify + deploy (FR-010)
- `go build ./...`, `go vet ./...`, `gofmt -l`, `go test ./... -count=1`.
- `task install` (rebuild `goal` → GOBIN). Note for the user to reload the VSCode window;
  confirm no `editors/vscode/package.json` change needed.
- *Depends on*: Steps 1–6.

## Sequencing
Steps 1 and 2 are independent; 3 and 4 depend on 1; 5 depends on 3+4; 6 on all; 7 last.

## Open items carried to tasks
- OQ-1 (unfixable `reports` as info diagnostics) — deferred to a follow-up.
- OQ-2 (enum-variant / struct-field child symbols) — v1 top-level only; cheap follow-up.
- Final helper/file names cosmetic; settle in tasks.
