# Implementation Plan — US-048 LSP go-to-definition

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/lsp/definition.go` | `definition` handler + AST symbol-graph resolution (declaration index, reference walk, position->offset). |
| `internal/lsp/definition_test.go` | AC tests: function-call and enum-variant resolution, type-name resolution, null fallbacks. |

### Modified Files
| File | Change |
|------|--------|
| `internal/lsp/protocol.go` | Add `Location`, `DefinitionParams`; add `ServerCapabilities.DefinitionProvider bool`. |
| `internal/lsp/server.go` | Advertise `DefinitionProvider: true` at initialize; route `textDocument/definition` to `s.definition`. |

## Design

### Types (protocol.go)
- `Location{URI string `json:"uri"`; Range Range `json:"range"`}`
- `DefinitionParams{TextDocument textDocumentIdentifier; Position Position}`
- `ServerCapabilities.DefinitionProvider bool `json:"definitionProvider,omitempty"`

### Handler (definition.go)
- `(s *Server) definition(raw) *Location`: decode params, fetch buffer (null on
  miss), call `resolveDefinition(src, line0, char0)`, wrap target Range in a
  `Location` with the request URI. Returns `nil` (marshals to JSON `null`) when
  unresolved. `reply` already marshals a nil pointer as `null`.

### Resolution (definition.go)
- `resolveDefinition(src, line0, char0) (Range, bool)`:
  1. `offsetForPosition(src, line0, char0)` -> byte offset (null on out-of-range).
  2. `parser.ParseFile(src)`; nil/err -> not found.
  3. `buildDeclIndex(file)` -> name-keyed declaration ranges.
  4. `collectRefs(src, file, idx)` -> `[]defRef{start,end,target Range}`.
  5. First ref whose `[start,end)` contains the offset -> `target, true`.
- `declIndex{funcs map[string]Range; types map[string]Range; variants map[string]map[string]Range}`
  built from `file.Decls`: FuncDecl (funcs), EnumDecl (types + variants),
  SealedInterfaceDecl (types), GenDecl TYPE TypeSpec (types).
- `collectRefs` uses `ast.Walk` with a visitor keyed by structural parent
  (mirrors `roleVisitor`): CallExpr.Fun ident -> funcs; CallExpr.Fun selector
  Sel -> funcs (method); SelectorExpr over a known enum -> enum (X) + variant
  (Sel); VariantLit / VariantPattern -> enum + variant; type-position idents
  (field/param/result/payload/alias) -> types.
- Ranges via the existing `rangeOf(src, startOff, endOff)`.

### offsetForPosition
Local helper: walk to the `line0`-th line start, add `char0`, clamp to len(src).
Inverse of `check.OffsetToPosition`. None exists in the tree.

## Requirement Traceability
- FR-1 -> server.go capability + `TestServerInitializeCapabilities`-style assert.
- FR-2 (call) -> CallExpr ref + AC test.
- FR-3 (variant) -> VariantLit/VariantPattern/SelectorExpr ref + AC test.
- FR-4 (type/enum name) -> type-position + selector-enum ref + test.
- FR-5 (null) -> nil returns + empty/unknown/unparseable tests.

## Testing
- `package lsp` internal test (stdlib `testing`, no testify), mirroring
  `semantictokens_test.go`: locate substrings via `strings.Index` +
  `check.OffsetToPosition` to form cursor positions, call `resolveDefinition`,
  assert the returned range maps back to the declaration's name offset.
- Reuse `testServer`/`fakeFiles`/`fakeResolver` for the handler test.

## Out of Scope
Cross-file resolution; locals/params/qualified symbols; hover/rename/references.
