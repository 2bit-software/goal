# Implementation Plan — US-050 LSP rename and find references

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/lsp/references.go` | The references + rename handlers and the inverted symbol-graph occurrence collector shared by both. |
| `internal/lsp/references_test.go` | AC tests: references for func/type/variant, includeDeclaration toggle, rename produces edits at every reference, null fallbacks, capability advertisement. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/lsp/protocol.go` | Add `ReferenceParams`, `ReferenceContext`, `RenameParams`; add `ReferencesProvider`/`RenameProvider` to `ServerCapabilities`. |
| `internal/lsp/server.go` | Advertise `referencesProvider:true` + `renameProvider:true` at initialize; route `textDocument/references` and `textDocument/rename`. |

## Package Structure

All within `internal/lsp/` — no new package. references.go sits beside
definition.go/hover.go and reuses their exported-within-package helpers
(`buildDeclIndex`, `offsetForPosition`, `rangeOf`, `identRange`, `named`/`name`).

## Dependency Graph

1. protocol.go types (no deps).
2. references.go occurrence collector — reuses definition.go's `declIndex` /
   `buildDeclIndex` and the structural reference keying; produces keyed
   occurrences `{start,end,key,isDecl}`.
3. references.go handlers `references` / `rename` (depend on 2 + 1).
4. server.go routing + capability advertisement (depends on 1, 3).
5. references_test.go (depends on all).

## Algorithm

- `collectOccurrences(src, file, idx)`: seed declaration-name occurrences
  (isDecl=true) for funcs/types/enum+variants, then walk the AST with an
  occurrence visitor mirroring definition.go's `refVisitor` structural keying
  (CallExpr.Fun ident/method-sel, SelectorExpr over a known enum, VariantLit/
  VariantPattern, type-position idents), recording each as `{start,end,key}`.
- A `symKey{kind, enum, name}` identifies a symbol: func / type / variant
  (variant carries its enum).
- `resolveOccurrences(src, line, char)`: parse, build index, collect
  occurrences, find the one covering the cursor offset, and return all
  occurrences sharing its key (plus the matched key).
- `references`: map same-key occurrences to `Location`s; drop isDecl ones when
  `context.includeDeclaration` is false; nil -> JSON null.
- `rename`: validate newName is a legal identifier; map same-key occurrences to
  `TextEdit{rangeOf(span), newName}`; wrap in `WorkspaceEdit{DocumentChanges:
  [TextDocumentEdit{versioned URI, edits}]}`; nil -> JSON null.

## Testing

stdlib `testing` only. Reuse `offsetOfNth` / `cursorAt` helpers from
definition_test.go (same package). Drive `resolveOccurrences` directly and the
handlers via marshalled params + `s.upsert`, mirroring TestDefinitionHandler.
