# Research Findings — US-049 LSP hover

This is an internal feature extending the existing goal LSP; no external research
was required. Findings come from reading the codebase.

## Existing patterns to reuse (High confidence)

- `internal/lsp/definition.go` (US-048) is the template: parse the open buffer
  with `parser.ParseFile`, build a name-keyed declaration index over
  `file.Decls`, walk references with a visitor keyed by structural parent
  (`refVisitor`), and map the cursor to a byte offset with `offsetForPosition`
  (the inverse of `check.OffsetToPosition`). Hover follows the same shape but
  maps a covering reference/declaration span to the symbol's rendered
  signature + doc rather than to a declaration `Range`.
- `internal/lsp/semantictokens.go` `roleVisitor` shows the structural-parent
  keying for enum/variant/type/func references.
- Best-effort contract (null on unknown URI / no symbol / unparse) is shared by
  `definition`, `documentSymbol`, and `semanticTokens`; hover must match it (a
  nil `*Hover` marshals to JSON `null`).

## Signature source (High confidence)

- `ast.FuncDecl.Pos().Offset` → `ast.FuncDecl.Type.End().Offset` is the raw
  source span of the full signature (modifier + `func` + name + params +
  results), excluding the body. Whitespace-collapsing it yields a clean
  one-line signature that includes a `Result[...]`/`Option[...]` result type
  verbatim — satisfying the AC for a Result-returning function.
- `ast.FuncDecl.Doc *ast.DocComment` carries the `///` lines (already
  prefix-stripped). Enum/sealed/type/variant nodes do not carry doc in the AST,
  so doc rendering applies to functions/methods.

## LSP protocol (High confidence)

- `textDocument/hover` → `Hover{ contents: MarkupContent{ kind, value } }`.
  Advertise `hoverProvider: true`. Standard, well-documented LSP shape.

## Confidence: High. No open questions. Next: spec + implement.
