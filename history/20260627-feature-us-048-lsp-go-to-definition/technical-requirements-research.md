# Technical Requirements / Research — US-048

## Constraints

- Zero-dependency: stdlib `testing` only (no testify).
- Derive everything from the AST (REWRITE-ARCHITECTURE.md): parse with
  `parser.ParseFile`, build a name-keyed declaration index, and resolve the
  identifier under the cursor through it — no token re-scan for findings.

## Approach

- Mirror the existing AST-backed LSP features (`internal/lsp/symbols.go`,
  `semantictokens.go`): parse + `ast.Walk`, keyed by structural parent node.
- Two passes over the parsed `ast.File`:
  1. Build a declaration index: function/method names, type/enum/sealed/alias
     names, and `Enum.Variant` -> declaration ranges.
  2. Walk references (CallExpr.Fun idents, SelectorExpr over a known enum,
     VariantLit/VariantPattern enum+variant, and type-position idents) recording
     each reference's byte span -> target declaration range.
- The handler converts the LSP 0-based position to a byte offset, finds the
  reference span containing it, and returns a `Location{URI, Range}` (the
  declaration's name range). Definition is within the open document (single-file
  AST symbol graph).

## Protocol additions

- `DefinitionParams{textDocument, position}` and `Location{uri, range}` types.
- `ServerCapabilities.DefinitionProvider bool`, advertised at initialize.
- Route `textDocument/definition` in `Server.handle`.

## Reused helpers

- `check.OffsetToPosition` for byte-offset -> 0-based range (via `rangeOf`).
- A new local `offsetForPosition(src, line0, char0)` for the inverse mapping
  (no such helper exists in the tree).
