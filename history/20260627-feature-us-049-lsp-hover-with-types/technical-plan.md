# Technical Plan — US-049 LSP hover with types

## Architecture

Add `textDocument/hover` to `internal/lsp`, reusing the US-048 AST symbol graph.
Hover resolves the symbol under the cursor (a reference OR a declaration name) to
a small `hoverInfo{ signature, doc }` and returns it as LSP `MarkupContent`.

## Files

### New: `internal/lsp/hover.go`

- `func (s *Server) hover(raw json.RawMessage) *Hover` — decode `HoverParams`,
  look up the open buffer (null if absent), call `resolveHover`, wrap the result
  as `*Hover` with markdown `MarkupContent`; nil → JSON null.
- `func resolveHover(src string, line, char int) (hoverInfo, bool)` — map cursor
  to offset via `offsetForPosition`, `parser.ParseFile`, build a hover index over
  `file.Decls`, collect covering spans (declaration names + references), return
  the `hoverInfo` whose `[start,end)` covers the offset.
- `type hoverInfo struct { signature, doc string }`
- `type hoverIndex struct { funcs map[string]hoverInfo; types map[string]hoverInfo;
  variants map[string]map[string]hoverInfo }` — name-keyed, variants under enum,
  mirroring `declIndex`.
- `buildHoverIndex(src, file)` — walk top-level decls; for `FuncDecl` render
  `funcSignature` + `docText(d.Doc)`; for `EnumDecl`/`SealedInterfaceDecl`/
  `TypeSpec` render a header; record each enum variant. Also append the
  declaration-name spans so hovering the declaration itself resolves.
- `collectHoverSpans(src, file, idx)` — walk the AST recording reference idents'
  byte spans → `hoverInfo`, keyed by structural parent exactly like
  `definition.go`'s `refVisitor` (call callee, enum selector, variant lit/pattern,
  type-position idents).
- `funcSignature(src, *ast.FuncDecl) string` — `strings.Join(strings.Fields(
  src[d.Pos().Offset:d.Type.End().Offset]), " ")`.
- `docText(*ast.DocComment) string` — join non-doctest `Lines` with newlines.

### Edit: `internal/lsp/protocol.go`

```go
type HoverParams struct {
    TextDocument textDocumentIdentifier `json:"textDocument"`
    Position     Position               `json:"position"`
}
type Hover struct {
    Contents MarkupContent `json:"contents"`
}
type MarkupContent struct {
    Kind  string `json:"kind"`  // "markdown"
    Value string `json:"value"`
}
// ServerCapabilities gains: HoverProvider bool `json:"hoverProvider,omitempty"`
```

### Edit: `internal/lsp/server.go`

- Advertise `HoverProvider: true` in the initialize capabilities.
- Route `case "textDocument/hover": s.reply(m.ID, s.hover(m.Params))`.

## Interface Contracts

- `resolveHover(src string, line, char int) (hoverInfo, bool)` — pure, testable
  without a server (mirrors `resolveDefinition`).
- `*Hover` nil ⇒ JSON `null` (best-effort contract).

## Integration Points

- Reuses `offsetForPosition` (definition.go), `parser.ParseFile`, `ast.Walk`,
  `check.OffsetToPosition` (already imported in the package).
- No change to existing handlers; additive only.

## Testing Strategy

New `internal/lsp/hover_test.go` (mirrors `definition_test.go`):

- `TestHoverResultFunction` — sample with a `func ... Result[...]`; assert the
  resolved signature string contains `Result` and the function name (AC-2).
- `TestHoverFunctionDoc` — function with a `///` doc; assert the doc text appears
  in the hover (AC-1).
- `TestHoverNoSymbol` / `TestHoverUnparseable` — null fallbacks.
- `TestHoverHandler` — open URI → non-nil Hover; unknown URI → nil.
- `TestServerAdvertisesHover` — initialize advertises `"hoverProvider":true`.

## Verification

`go build ./...`, `go vet ./...`, `go test ./... -count=1` (prd verifyCommands)
plus the new hover tests.
