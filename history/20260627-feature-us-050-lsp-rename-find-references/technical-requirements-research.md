# Technical requirements / research — US-050

## Existing seam to reuse

`internal/lsp/definition.go` (US-048) holds the AST symbol graph:

- `buildDeclIndex(src, file) declIndex` — name-keyed declaration table
  (funcs / types / variants[enum][tag] -> declaration Range).
- `refVisitor` / `collectRefs` — walks the AST recording each reference
  identifier's byte span, keyed by structural parent (CallExpr.Fun,
  SelectorExpr over a known enum, VariantLit/VariantPattern, type-position
  idents).
- `offsetForPosition(src, line, char)` — inverse of `check.OffsetToPosition`.
- `identRange` / `rangeOf` — protocol Range for an identifier span.

US-049 hover (`hover.go`) shows the pattern for ALSO seeding declaration-name
spans (so the cursor resolves when it sits on the declaration itself).

## Approach

References/rename invert the definition map. Instead of `ref -> declaration
Range`, build occurrences `{start, end, key, isDecl}` where `key` is a stable
symbol identity:

- func: `{kind:func, name}`
- type/enum/sealed/alias: `{kind:type, name}`
- enum variant: `{kind:variant, enum, name}` (keyed under its enum so a tag
  shared by two enums stays distinct, mirroring the declIndex).

Seed declaration-name occurrences (isDecl=true), then walk references with the
same structural keying used by `refVisitor`. Find the occurrence covering the
cursor offset, take its key, and return every occurrence sharing that key.

- `references`: `[]Location` for every same-key occurrence; honor
  `ReferenceContext.IncludeDeclaration` by filtering out `isDecl` occurrences
  when false.
- `rename`: a `WorkspaceEdit{DocumentChanges: [TextDocumentEdit{...}]}` with one
  `TextEdit{Range, NewText:newName}` per same-key occurrence. Validate the new
  name is a legal identifier (non-empty, letter/`_` then letter/digit/`_`); an
  invalid name yields null.

## Protocol additions (protocol.go)

- `ReferenceParams{TextDocument, Position, Context ReferenceContext}`,
  `ReferenceContext{IncludeDeclaration bool}`.
- `RenameParams{TextDocument, Position, NewName string}`.
- `ServerCapabilities.ReferencesProvider bool`, `RenameProvider bool`.
- `WorkspaceEdit` / `TextDocumentEdit` / `TextEdit` /
  `versionedTextDocumentIdentifier` already exist (code-action reuses them).

## Wiring

`server.go` advertises `referencesProvider:true` / `renameProvider:true` at
initialize and routes `textDocument/references` -> `s.references`,
`textDocument/rename` -> `s.rename`. Nil slice / nil `*WorkspaceEdit` marshal to
JSON null (best-effort contract).
