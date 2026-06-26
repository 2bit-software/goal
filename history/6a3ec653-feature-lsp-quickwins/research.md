---
status: complete
updated: 2026-06-26
---

# Technical Research: High-ROI LSP quick wins

## Executive Summary

Three LSP features, each reusing existing infrastructure with no new analysis: (1) an
idiomatize code action wrapping `internal/fix.File`, (2) precise diagnostic ranges from
`scan.Lex` token ends, (3) document symbols from a lexical pass over tokens. The JSON-RPC
layer already supports request/response generically; the VSCode extension forwards
codeAction/documentSymbol automatically once the server advertises the capabilities — no
extension change needed.

## Findings

### LSP server surface today
- `internal/lsp/server.go` `handle()` switches on method: replies to `initialize`/`shutdown`
  (via `s.reply(m.ID, result)`), handles `exit`, the three `didOpen/didChange/didClose`
  notifications, no-ops `initialized`/`$/setTrace`/`didSave`, and returns
  `codeMethodNotFound` for any other request with an ID.
- `ServerCapabilities` (protocol.go) advertises only `{"textDocumentSync": 1}`.
- `internal/lsp/jsonrpc.go` is generic: `rpcMessage{ID, Method, Params}`, `rpcResponse`,
  `s.reply(id, result)` marshals any result. **Adding request handlers = new `case`s that
  call `s.reply`.** No plumbing change required.
- `s.docs map[string]*doc` holds current buffer text+version per URI (mutex-guarded);
  `openFilesInDir`/overlay from the prior initiative exist but are not needed here (all three
  features are single-buffer).

### VSCode client (no change needed)
- `editors/vscode/package.json`: `vscode-languageclient ^9.0.1`;
  `extension.ts` constructs a `LanguageClient` with `documentSelector:[{language:"goal"}]` and
  spawns `goal lsp` over stdio. vscode-languageclient auto-registers the documentSymbol and
  codeAction features from advertised server capabilities, so advertising them is sufficient.

### Feature 1 — idiomatize code action
- `internal/fix.File(src string) (out string, changes []fix.Change, reports []fix.Report)`
  runs every fixer to a fixed point; `out == src` when nothing changed. Single-file, no
  package context, already tested (`internal/fix/fix_test.go`).
- Delivery: handle `textDocument/codeAction`. If `out != src` for the request's URI buffer,
  return one `CodeAction{title: "Idiomatize file (goal fix)", kind: "source.fixAll.goal",
  edit}`. **Version-pinned edit** (audit C1): use `WorkspaceEdit{DocumentChanges:
  []TextDocumentEdit{{TextDocument: {URI, Version}, Edits: [TextEdit{Range, NewText: out}]}}}`
  where `Version` is the buffer's current `s.docs[uri].version`. Full-document range:
  start `{0,0}`; end = `check.OffsetToPosition(src, len(src))` converted to 0-based
  (`{Line-1, Col-1}`), computed from the SAME `src` passed to `fix.File`. Empty result (`[]`)
  otherwise. Honor `context.only` per FR-001b (prefix match).
- Why a code action (not `workspace/executeCommand`): a CodeAction carrying a WorkspaceEdit
  is applied by the client directly — works for manual Quick Fix AND
  `editor.codeActionsOnSave` with `source.fixAll`. No server→client `applyEdit` round-trip.
- `CodeActionParams` carries `textDocument.uri`, a `range`, and a `context` (with requested
  `only` kinds + current diagnostics). For a `source`-kind action we ignore the range and act
  on the whole document; honor `context.only` if the client filters kinds.
- `reports` (Skip/Warn unfixable candidates) — OQ-1; default deferred.

### Feature 2 — precise diagnostic ranges
- `toLSP(text, d)` (diagnostics.go) currently sets `endChar = lineLength(...)` because no
  token length is known. The check `Diagnostic.Pos` is a token `Start` offset; `scan.Lex(text)`
  yields tokens with `Start`/`End`. Build a `map[int]int` (start→end) once per file and pass
  it (or a small lookup) into `toLSP`; convert `End` via `check.OffsetToPosition`.
- Fallback: when `Pos` is not a token start (defensive), keep `lineLength` behavior.
- Touch points: `compile` (per-file publish loop) and `compileSingle` both call `toLSP`.
  Pinned signature change: `toLSP(text string, tokEnd map[int]int, d check.Diagnostic)`,
  where `tokEnd` is `tok.Start → tok.End` built once per file from `scan.Lex(text)`. Fallback
  to `lineLength` when `d.Pos ∉ tokEnd` or resolved end ≤ start (FR-004).
- `check.OffsetToPosition(src, off) check.Position{Line, Col}` is 1-based; `toLSP` already
  decrements to 0-based. End position uses the same conversion.

### Feature 3 — document symbols
- Handle `textDocument/documentSymbol`; advertise `documentSymbolProvider: true`.
- Extract top-level decls by a lexical pass over `scan.Lex` tokens (mirroring how
  `analyze`/`check` locate decls, but emitting positions which the tables discard):
  - `enum NAME { … }` → `SymbolKind.Enum`; variants → children (EnumMember) [OQ-2].
  - `type NAME struct { … }` → `Struct`; fields → children (Field) [OQ-2].
  - `type NAME interface { … }` / `sealed interface NAME` → `Interface`.
  - `type NAME = …` / other type decl → `Class`.
  - `func NAME(...)` → `Function`; `func (r R) NAME(...)` → `Method`.
  - `from func NAME` / `derive func NAME` → `Function` (detail shows the construct).
- **Do NOT route through `scan.ScanFuncs`** (audit correction): its `Func{Name, NameTok,
  ParamsClose, BodyOpen, BodyClose}` carries **no receiver flag, no `func`-keyword token
  index, and no `from`/`derive` visibility** — it cannot distinguish Method from Function nor
  give the keyword-start for the range. Write a dedicated token walk instead. Pinned API:
  `func documentSymbols(src string) []DocumentSymbol` in **`internal/lsp/symbols.go`** (pure,
  best-effort, never errors/panics per FR-006). The walk, over `scan.Lex(src)`:
  - at a `func` token `i`: it's a **Method** iff `toks[i+1].Text == "("` (receiver), else a
    **Function**; it's a `from`/`derive` func iff `toks[i-1].Text` is `from`/`derive` (kind
    Function, detail = the keyword). Range start = the keyword token's `.Start` (the
    `from`/`derive`/`func` token); `Name`/`SelectionRange` from the name identifier token.
  - at `enum`/`sealed`/`type` tokens: read the following name; range = keyword `.Start` →
    `scan.MatchBrace`'s close `.End` (bodyless `type X = …` → end of the decl line).
  - `SelectionRange` = name token `[Start,End)`, guaranteed ⊆ `Range`.
- `SymbolKind` integer values (LSP wire, pin in protocol.go): Class=5, Method=6, Field=8,
  Enum=10, Interface=11, Function=12, EnumMember=22, Struct=23.
- Return `DocumentSymbol[]` (hierarchical) — vscode-languageclient 9 advertises
  `hierarchicalDocumentSymbolSupport`, so this is supported and preferred.

### Protocol types to add (protocol.go)
- `CodeActionParams{TextDocument, Range, Context{Only []string, Diagnostics []Diagnostic}}`.
- `CodeAction{Title, Kind, Edit *WorkspaceEdit}`; `WorkspaceEdit{Changes map[string][]TextEdit}`;
  `TextEdit{Range, NewText}`.
- `DocumentSymbolParams{TextDocument}`; `DocumentSymbol{Name, Detail, Kind int, Range,
  SelectionRange, Children []DocumentSymbol}`; `SymbolKind` constants.
- Extend `ServerCapabilities` with `codeActionProvider` (object declaring
  `codeActionKinds: ["source.fixAll", "source.fixAll.goal"]`) and `documentSymbolProvider: true`.

## Decision Points

- [x] **D1 — Idiomatize as CodeAction+WorkspaceEdit**, not executeCommand (simpler, on-save capable).
- [x] **D2 — Version-pinned full-document replace** (`documentChanges` + `TextDocumentEdit`), range from `OffsetToPosition(src,len(src))`.
- [ ] **OQ-1 reports as diagnostics**: defer to follow-up (default); revisit in plan.
- [ ] **OQ-2 symbol children**: top-level first; add enum-variant/struct-field children if cheap.
- [x] **D3 — Range mapping via per-file start→end token map** (`toLSP(text, tokEnd, d)`), fallback to line end.
- [x] **D4 — No extension change**: advertise capabilities; vscode-languageclient auto-registers.
- [x] **D5 — Symbols via a dedicated token walk** (`documentSymbols(src) []DocumentSymbol` in symbols.go), NOT `scan.ScanFuncs` (no receiver/keyword info). Method = `func` followed by `(`; from/derive = preceding keyword.
- [x] **D6 — Deploy on completion**: `task install` + reload VSCode (FR-010); no `package.json` change.

## Recommendations

1. Add request handlers `textDocument/codeAction` and `textDocument/documentSymbol` to
   `handle()`, each reading the buffer from `s.docs` and replying via `s.reply`.
2. Add the protocol types + capability advertisements.
3. Precise ranges: thread a token start→end lookup into `toLSP`.
4. Symbols: a dedicated `symbols.go` lexical pass reusing `scan` + `check.OffsetToPosition`.
5. Test with the existing synchronous-server harness; unit-test range mapping and symbol
   extraction directly. Rebuild/install `goal` and reload VSCode to pick up the new server.

## Risks / Watch-items

- Code-action `context.only` filtering: if the client asks for specific kinds, only return the
  fix-all action when its kind is requested (or when `only` is empty).
- Full-document edit range must cover the exact current document extent (use last line/col of
  the buffer the action was computed from) to avoid truncation.
- Symbol ranges must stay within the document; clamp via `OffsetToPosition` (already clamps).
- Don't run `fix.File` on every keystroke — only on code-action request (and, if OQ-1 adopted,
  on `didSave`).
- Code-action/symbol handlers must not deadlock with `analysisMu` (they don't touch it).

## Sources
- `internal/lsp/{server.go,diagnostics.go,jsonrpc.go,protocol.go,server_test.go}`
- `internal/fix/fix.go` (`File`, `Change`, `Report`, `Level`)
- `internal/scan/scan.go` (`Token{Start,End}`, `Lex`, `ScanFuncs`, `Func`, `MatchBrace`)
- `internal/check/check.go` (`OffsetToPosition`, `Position`)
- `editors/vscode/{package.json,src/extension.ts}`
