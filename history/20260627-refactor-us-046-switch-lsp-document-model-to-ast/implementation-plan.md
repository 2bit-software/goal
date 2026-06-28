# Implementation Plan — US-046 Switch LSP document model to AST

## File Inventory

### New Files
None.

### Modified Files
| File | Changes |
|------|---------|
| `internal/lsp/symbols.go` | Rewrite `collectSymbols` to derive the outline from `parser.ParseFile` over `internal/ast`. Remove the `decl` struct, `scanDecls`, `skipLine`, `declEnd`, and the `internal/scan` import. Add a declaration-walk that maps each top-level `ast.Decl` to a `DocumentSymbol`. |
| `internal/lsp/diagnostics.go` | Replace `scan.Lex` in `tokenEnds` with `lexer.Tokens`; token end = `Pos.Offset + len(Lit)`. Drop the `internal/scan` import (switch to `internal/lexer`). |

## Dependency Graph

1. Rewrite `symbols.go` `collectSymbols` onto `parser.ParseFile` (no deps).
2. Switch `diagnostics.go` `tokenEnds` onto `lexer.Tokens` (no deps).
3. Run `internal/lsp` tests + project verify gates.

## Interface Contracts

Unchanged public/internal seams:

```go
// symbols.go — signature preserved so tests compile unchanged.
func collectSymbols(src string) []DocumentSymbol

// diagnostics.go — signature preserved.
func tokenEnds(text string) map[int]int
```

New internal helper in symbols.go:

```go
// symbolFor maps one top-level declaration to a DocumentSymbol, or ok=false to skip.
func symbolFor(src string, d ast.Decl) (DocumentSymbol, bool)
```

Node -> (kind, name, detail):
- `*ast.EnumDecl`            -> symEnum,      d.Name, ""
- `*ast.SealedInterfaceDecl` -> symInterface, d.Name, "sealed interface"
- `*ast.GenDecl{Tok:TYPE}`   -> one symbol per `*ast.TypeSpec`:
    - `spec.Assign != zero`            -> symClass  (alias)
    - `spec.Type` is `*ast.StructType` -> symStruct
    - `*ast.InterfaceType`             -> symInterface
    - otherwise                        -> symClass
  Range start = GenDecl.Pos() (the `type` keyword) when the decl has a single
  spec; otherwise spec.Pos(). Range end = spec.End().
- `*ast.FuncDecl`            -> symMethod if Recv != nil else symFunction
  (from/derive stay symFunction; Pos() already points at the modifier keyword).

Range = node Pos()..End() offsets -> `rangeOf`. SelectionRange = name Pos()..End().
A declaration whose name Ident is nil/empty is skipped (best-effort).

## Integration Points

- `documentSymbols(raw json.RawMessage)` is unchanged; it still calls
  `collectSymbols(text)`.
- `compile`/`compileSingle` are unchanged; they still call `tokenEnds(text)`
  and `toLSP`.
- Offset->Position conversion stays `check.OffsetToPosition` via `rangeOf`.

## Testing Strategy

- The existing `internal/lsp` tests are the contract — run the whole package:
  `go test ./internal/lsp -count=1`.
- Project verify gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
- Grep gate: `scanDecls` no longer present in `internal/lsp`; `internal/scan`
  no longer imported by `internal/lsp`.
