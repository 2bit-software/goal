# Technical Spec: High-ROI LSP quick wins

Implementation design. Reuses `internal/fix.File`, `scan.Lex`/scan helpers, and
`check.OffsetToPosition`. Stdlib-only. No VSCode extension change. See `research.md` for the
pinned decisions (D1–D6) and `spec.md` for FRs.

## Change set

| Area | File | Change |
|---|---|---|
| Protocol types + capabilities | `internal/lsp/protocol.go` | CodeAction*, WorkspaceEdit/TextDocumentEdit/TextEdit, DocumentSymbol*, SymbolKind consts; extend ServerCapabilities |
| Capability advertisement | `internal/lsp/server.go` | initialize result advertises codeAction + documentSymbol providers |
| Request handlers | `internal/lsp/server.go` | `handle()` cases for `textDocument/codeAction`, `textDocument/documentSymbol` |
| Code action logic | `internal/lsp/codeaction.go` (new) | `fix.File` → version-pinned fixAll edit; `context.only` filter |
| Symbol extraction | `internal/lsp/symbols.go` (new) | `documentSymbols(src) []DocumentSymbol` (depth-0 token walk) |
| Precise ranges | `internal/lsp/diagnostics.go` | `toLSP(text, tokEnd, d)`; build `tokEnd` per file at both call sites |
| Tests | `internal/lsp/*_test.go` | codeAction, documentSymbol, range, fallback, unknown-URI |

## 1. Protocol types (protocol.go)

```go
// Capabilities (extend ServerCapabilities)
type ServerCapabilities struct {
    TextDocumentSync       int                  `json:"textDocumentSync"`
    CodeActionProvider     *CodeActionOptions   `json:"codeActionProvider,omitempty"`
    DocumentSymbolProvider bool                 `json:"documentSymbolProvider,omitempty"`
}
type CodeActionOptions struct {
    CodeActionKinds []string `json:"codeActionKinds,omitempty"` // ["source.fixAll","source.fixAll.goal"]
}

// codeAction request
type CodeActionParams struct {
    TextDocument textDocumentIdentifier `json:"textDocument"`
    Range        Range                  `json:"range"`
    Context      CodeActionContext      `json:"context"`
}
type CodeActionContext struct {
    Only        []string     `json:"only,omitempty"`
    Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}
type CodeAction struct {
    Title string        `json:"title"`
    Kind  string        `json:"kind,omitempty"`
    Edit  *WorkspaceEdit `json:"edit,omitempty"`
}
// Version-pinned edit (FR-001a)
type WorkspaceEdit struct {
    DocumentChanges []TextDocumentEdit `json:"documentChanges"`
}
type TextDocumentEdit struct {
    TextDocument versionedTextDocumentIdentifier `json:"textDocument"` // {uri, version}
    Edits        []TextEdit                      `json:"edits"`
}
type TextEdit struct {
    Range   Range  `json:"range"`
    NewText string `json:"newText"`
}

// documentSymbol request
type DocumentSymbolParams struct {
    TextDocument textDocumentIdentifier `json:"textDocument"`
}
type DocumentSymbol struct {
    Name           string           `json:"name"`
    Detail         string           `json:"detail,omitempty"`
    Kind           int              `json:"kind"`
    Range          Range            `json:"range"`
    SelectionRange Range            `json:"selectionRange"`
    Children       []DocumentSymbol `json:"children,omitempty"`
}

// SymbolKind wire values (LSP)
const (
    symClass     = 5
    symMethod    = 6
    symField     = 8
    symEnum      = 10
    symInterface = 11
    symFunction  = 12
    symEnumMember = 22
    symStruct    = 23
)
```

Initialize advertises:
```go
Capabilities: ServerCapabilities{
    TextDocumentSync:       fullSync,
    CodeActionProvider:     &CodeActionOptions{CodeActionKinds: []string{"source.fixAll", "source.fixAll.goal"}},
    DocumentSymbolProvider: true,
},
```

## 2. handle() request routing (server.go)

Add two cases (both reply via `s.reply(m.ID, result)`; both read the buffer from `s.docs`
under `s.mu`; neither touches `analysisMu`, so no deadlock — FR-009):
```go
case "textDocument/codeAction":
    s.reply(m.ID, s.codeActions(m.Params))     // returns []CodeAction (never nil → [])
case "textDocument/documentSymbol":
    s.reply(m.ID, s.symbols(m.Params))          // returns []DocumentSymbol (never nil → [])
```
Helper to fetch buffer: `func (s *Server) buffer(uri string) (text string, version int, ok bool)`
locking `s.mu`. Unknown/closed URI → `ok=false` → handler returns `[]` (FR-007). Use
`out := []T{}` (non-nil) so JSON marshals `[]`, not `null`.

## 3. Code action (codeaction.go)

```go
func (s *Server) codeActions(raw json.RawMessage) []CodeAction {
    var p CodeActionParams
    if !s.decode(raw, &p, "codeAction") { return []CodeAction{} }
    if !wantsKind(p.Context.Only, "source.fixAll.goal") { return []CodeAction{} } // FR-001b
    text, version, ok := s.buffer(p.TextDocument.URI)
    if !ok { return []CodeAction{} }
    out, _, _ := fix.File(text)
    if out == text { return []CodeAction{} } // no-op → no action (FR-001)
    end := check.OffsetToPosition(text, len(text))
    return []CodeAction{{
        Title: "Idiomatize file (goal fix)",
        Kind:  "source.fixAll.goal",
        Edit: &WorkspaceEdit{DocumentChanges: []TextDocumentEdit{{
            TextDocument: versionedTextDocumentIdentifier{URI: p.TextDocument.URI, Version: version},
            Edits: []TextEdit{{
                Range:   Range{Start: Position{0, 0}, End: Position{Line: end.Line - 1, Character: end.Col - 1}},
                NewText: out,
            }},
        }}},
    }}
}

// wantsKind reports whether the client's `only` filter admits kind (empty ⇒ yes; else any
// requested entry must be a prefix of kind by LSP hierarchical rules: "source",
// "source.fixAll", "source.fixAll.goal").
func wantsKind(only []string, kind string) bool {
    if len(only) == 0 { return true }
    for _, o := range only {
        if o == kind || strings.HasPrefix(kind, o+".") { return true } // "source" matches "source.fixAll.goal"
    }
    return false
}
```
`source.fixAll` is kept in the advertised `CodeActionKinds` as an umbrella so a user's
`editor.codeActionsOnSave: {"source.fixAll": true}` triggers our `source.fixAll.goal` action
(matched by `wantsKind`); the handler always emits the specific `source.fixAll.goal` kind.

`fix.File` is lexical/conservative and never panics on partial source (FR-009); a defensive
`recover` is unnecessary but a `// fix.File is total over any string` note suffices.

## 4. Document symbols (symbols.go)

`func documentSymbols(src string) []DocumentSymbol` — best-effort, never panics (FR-006).

**Two-phase to bound ranges safely (audit M1/M2/m3 fix).** A single forward `{`-scan is unsafe:
a bodyless `from`/`derive func` or a `type X = …` alias would find the *next* declaration's
brace and swallow it (`FirstBodyBrace`/`ParamsClose` scan unbounded). Both bodied AND bodyless
`from`/`derive func` exist in the corpus, so body-presence must be detected, not assumed.

- **Phase 1** — walk `scan.Lex(src)` tracking nesting via `{}()[]`; at depth 0 record each
  top-level decl as `{kind, kwTok, nameTok}`:
  - `enum` → `symEnum`, name = next ident.
  - `sealed`+`interface` → `symInterface`, name = ident after `interface`.
  - `type` → name = next ident; kind from the token *after the name*: `struct`→`symStruct`,
    `interface`→`symInterface`, `=`→`symClass` (alias), else `symClass`. (Alias-ness comes
    from the literal `=` token — never a forward brace scan.)
  - `func` (guard `i>0` before reading `toks[i-1]`) → kwTok = `i-1` if `toks[i-1]` is
    `from`/`derive` (Detail "from func"/"derive func"), else `i`. **Method** iff
    `toks[i+1].Text=="("` → `symMethod`, name = ident after `MatchParen(toks, i+1)`; else
    `symFunction`, name = `toks[i+1]`.
  - `from`/`derive`/`sealed`/`struct` are plain identifiers matched by `.Text`; the `from`/
    `derive` tokens themselves are not switched on (handled at the following `func`).
- **Phase 2** — for decl k, its declaration span is bounded by the next decl's `kwTok`
  (or EOF): `limit = nextDecl.kwTok` (else `len(toks)`). Body brace = the first `{` strictly
  before `limit`. If present → `Range` end = `MatchBrace(thatBrace)`.End (always present and
  in-span for enum/struct/interface/sealed and bodied funcs). If absent → bodyless decl
  (bodyless func / alias) → `Range` end = `scan.NextNewline(src, sigStart)` where `sigStart`
  is the keyword start offset (end of the signature's line). `Range` start = `kwTok`.Start.
- `SelectionRange` = name token `[Start,End)` (⊆ Range). Convert all offsets via
  `check.OffsetToPosition` (1-based) → 0-based `lsp.Position{Line-1, Col-1}`.

OQ-2 (children): v1 emits top-level symbols only; enum variants / struct fields as `Children`
is a cheap follow-up (walk the body brace span) but not required for the acceptance test.

Note: `scan.ScanFuncs` is intentionally NOT used — it requires a body brace (drops bodyless
`from`/`derive func`) and exposes neither receiver nor keyword position. `FirstBodyBrace`/
`ParamsClose` are NOT used for ranges (unbounded — the M1/M2 bug); the next-decl bound above
replaces them.

## 5. Precise ranges (diagnostics.go)

`toLSP(text string, tokEnd map[int]int, d check.Diagnostic) Diagnostic`:
- `start := check.OffsetToPosition(text, d.Pos)`.
- if `endOff, ok := tokEnd[d.Pos]; ok` → `endP := check.OffsetToPosition(text, endOff)`;
  set End from `endP` (0-based). Else (or if End ≤ Start on the same line) → today's
  `lineLength` fallback (FR-004).
- Build `tokEnd` once per file: `for _, t := range scan.Lex(text) { m[t.Start] = t.End }`.
  In `compile`, build it per `views[i].src`; in `compileSingle`, build it from `text`. Pass in.

## Test plan (stdlib testing)

Reuse the synchronous harness (`NewServerWithIO`, `debounce<=0`, framed-message capture,
direct `s.didOpen`). Drive requests by calling `s.codeActions(raw)` / `s.symbols(raw)`
directly (unit-ish) and via the framed `handle` path (integration):
- codeAction: fixable file → one `source.fixAll.goal` action whose edit's `NewText` == `fix.File` out and `Version` == buffer version; no-op file → `[]`; `only` filter cases; unknown URI → `[]`; broken-source-with-fixable-pattern → no panic.
- documentSymbol: fixture with enum/struct/interface/sealed/alias/func/method/from/derive → each present once with correct kind + ranges; empty file → `[]`; partial source → best-effort, no panic.
- toLSP: range uses token end; non-token offset & end≤start → fallback; multi-line token end.
- initialize advertises both providers.
- Existing diagnostics tests still green (FR-008).

## Deploy (FR-010)
`task install` (rebuild `goal` → GOBIN), then reload the VSCode window. No `package.json`
change. SC-003 outline population verified manually post-reload.
