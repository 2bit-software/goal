# Technical Plan — US-047 LSP semantic tokens

## Files

### internal/lsp/protocol.go (edit)
Add wire types + legend:
- `SemanticTokensOptions{ Legend SemanticTokensLegend; Full bool }` (json:
  `legend`, `full`).
- `SemanticTokensLegend{ TokenTypes []string; TokenModifiers []string }` (json:
  `tokenTypes`, `tokenModifiers`).
- `SemanticTokensParams{ TextDocument textDocumentIdentifier }`.
- `SemanticTokens{ Data []uint }` (json: `data`).
- Extend `ServerCapabilities` with
  `SemanticTokensProvider *SemanticTokensOptions json:"semanticTokensProvider,omitempty"`.
- Legend index constants (`semKeyword`, `semType`, `semEnum`, `semInterface`,
  `semStruct`, `semParameter`, `semVariable`, `semProperty`, `semEnumMember`,
  `semFunction`, `semMethod`, `semString`, `semNumber`, `semComment`,
  `semOperator`) and the ordered `semanticTokenTypes` / `semanticTokenModifiers`
  slices that back the legend (order MUST match the index constants).

### internal/lsp/semantictokens.go (new)
- `func (s *Server) semanticTokens(raw json.RawMessage) SemanticTokens` — decode
  params, fetch the open buffer (empty `SemanticTokens{Data: []uint{}}` when the
  URI is unknown), delegate to `computeSemanticTokens`.
- `func computeSemanticTokens(src string) []uint`:
  1. `roles := astRoles(src)` — byte-offset -> legend-index override map from the
     AST (best effort; parse failure yields an empty map).
  2. `toks := lexer.Tokens(src)`.
  3. For each token, `classifyToken(t, roles)` returns `(semType int, ok bool)`.
     Skip `ok==false`. Compute `length` = `len(t.Lit)` or `len(t.Kind.String())`
     when `Lit==""`. Convert `t.Pos` to 0-based line/char via the existing
     `check.OffsetToPosition`.
  4. Delta-encode in document order into `[]uint` 5-tuples.
- `func classifyToken(t token.Token, roles map[int]int) (int, bool)`:
  - comments (`COMMENT`,`DOC_COMMENT`) -> semComment
  - `STRING`,`CHAR` -> semString
  - `INT`,`FLOAT`,`IMAG` -> semNumber
  - `t.Kind.IsKeyword()` -> semKeyword (covers `match`, `enum`, etc.)
  - `QUESTION`,`FAT_ARROW`,`ELLIPSIS` -> semOperator
  - `IDENT` -> `roles[offset]` if present, else not emitted
  - otherwise not emitted (delimiters / arithmetic)
- `func astRoles(src string) map[int]int`: `parser.ParseFile`; on error/nil
  return empty map; else `ast.Walk` a `roleVisitor` that records, by
  `ident.Pos().Offset`:
  - EnumDecl.Name -> semEnum; each Variant.Name -> semEnumMember; payload field
    Name -> semProperty
  - SealedInterfaceDecl.Name -> semInterface
  - TypeSpec.Name -> semStruct (StructType) / semInterface (InterfaceType) /
    semType (alias / other)
  - FuncDecl.Name -> semMethod when Recv!=nil else semFunction; receiver/param/
    result Field.Names -> semParameter
  - VariantLit / VariantPattern: `.Variant` -> semEnumMember; `.Enum` (when
    *Ident) -> semEnum; VariantPattern.Binding -> semVariable
  - CallExpr.Fun (when *Ident) -> semFunction (only if not already set)

### internal/lsp/server.go (edit)
- In `handle`, advertise `SemanticTokensProvider: &SemanticTokensOptions{Legend:
  defaultSemanticLegend(), Full: true}` in the `initialize` reply.
- Add case `"textDocument/semanticTokens/full": s.reply(m.ID, s.semanticTokens(m.Params))`.

## Integration Points
- Reuses `lexer.Tokens`, `parser.ParseFile`, `ast.Walk`, `check.OffsetToPosition`
  — all already imported elsewhere in internal/lsp or its deps. No new external
  dependency.
- Mirrors the structure of `internal/lsp/symbols.go` (US-046).

## Testing Strategy
New `internal/lsp/semantictokens_test.go` (package lsp, stdlib testing):
- `TestComputeSemanticTokensEnumMatchQuestion`: a sample with an enum, a match,
  and a `?`; decode the delta-encoded data back to absolute (line,char,len,type)
  tuples via a helper, and assert: the enum name -> semEnum, variant names ->
  semEnumMember, `match` -> semKeyword, `?` -> semOperator.
- `TestSemanticTokensWellFormed`: data length is a multiple of 5 and deltas are
  non-decreasing in document order.
- `TestSemanticTokensEmptyAndUnparseable`: empty source and a broken document
  yield a non-nil empty data slice, no panic.
- `TestSemanticTokensHandler`: handler returns tokens for an open URI and empty
  for an unknown URI.
- `TestServerAdvertisesSemanticTokens` (in server_test.go style): initialize
  response contains `"semanticTokensProvider"`.
