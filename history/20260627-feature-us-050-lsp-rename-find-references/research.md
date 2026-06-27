# Research — US-050 LSP rename and find references

No external research required: this is an internal extension of the goal LSP's
existing AST symbol graph. The authoritative references are in-repo.

## LSP protocol shape (from the LSP spec, already modeled in protocol.go)

- `textDocument/references` request -> `ReferenceParams{textDocument, position,
  context:{includeDeclaration:bool}}`; response is `Location[]` or `null`.
- `textDocument/rename` request -> `RenameParams{textDocument, position,
  newName}`; response is a `WorkspaceEdit` or `null`. The version-pinned
  `documentChanges` form (already used by the idiomatize code action) lets the
  client reject a stale edit.

## In-repo precedent (the pattern to mirror)

- `internal/lsp/definition.go` — `buildDeclIndex`, `refVisitor`/`collectRefs`,
  `offsetForPosition`, `identRange`/`rangeOf`. The structural keying of
  references (CallExpr.Fun, SelectorExpr over a known enum, VariantLit/
  VariantPattern, type-position idents) is the exact set of reference sites
  rename/references must also touch — so reusing it guarantees parity with
  go-to-definition coverage.
- `internal/lsp/hover.go` — precedent for ALSO seeding declaration-name spans so
  the cursor resolves on the declaration itself.
- `internal/lsp/codeaction.go` — precedent for constructing a `WorkspaceEdit`
  with `DocumentChanges` and the buffer version from `s.buffer(uri)`.

## Decision

Invert the definition map into keyed occurrences and return same-key
occurrences. No new third-party code, no alternative library to weigh — the
build/vet/test gates are the only verification surface.
