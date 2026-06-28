# US-050 LSP rename and find references — Business Specification

## Overview

The goal language server already resolves a symbol under the cursor to its
declaration (go-to-definition, US-048) and describes it (hover, US-049). This
feature adds the two refactoring operations that invert that symbol graph:
**find references** lists every place a symbol is used, and **rename** rewrites a
symbol and all of its uses in one atomic edit. Both operate over the AST symbol
graph of the single open document.

## Functional Requirements

### FR-1: Find references
`textDocument/references` returns every occurrence of the symbol under the
cursor — the function/method, type (enum/sealed/struct/alias), or enum variant —
as a list of `Location`s within the document. The set covers the declaration
name and every reference site reachable through the AST symbol graph (the same
structural reference set go-to-definition resolves).

### FR-2: Include-declaration honored
The request's `context.includeDeclaration` flag controls whether the
declaration's own name occurrence is part of the returned set. When false, only
reference sites are returned.

### FR-3: Rename
`textDocument/rename` returns a `WorkspaceEdit` whose `documentChanges` carry one
`TextEdit` per occurrence of the symbol under the cursor (declaration name plus
every reference), each replacing the identifier span with the new name. The edit
is version-pinned to the current buffer.

### FR-4: Variant/type distinctness
A rename or reference query on an enum variant affects only that variant's
occurrences, not the enum type, and a variant tag shared by two enums stays
distinct (variants are keyed under their enclosing enum).

### FR-5: Capability advertisement
Initialize advertises `referencesProvider:true` and `renameProvider:true`.

## Acceptance Criteria

- [ ] `textDocument/references` returns all reference sites for a function, a
      type/enum name, and an enum variant via the AST symbol graph.
- [ ] `includeDeclaration:false` omits the declaration occurrence;
      `includeDeclaration:true` includes it.
- [ ] `textDocument/rename` returns a `WorkspaceEdit` with a `TextEdit` at every
      reference (plus the declaration) of the symbol under the cursor.
- [ ] A test asserts renaming a symbol produces edits at every reference in a
      sample.
- [ ] An unknown URI, a cursor over no resolvable symbol, unparseable source, or
      (for rename) an invalid new name yields a null result and no panic.
- [ ] Initialize advertises `referencesProvider` and `renameProvider`.
- [ ] `go build ./...`, `go vet ./...`, `go test ./... -count=1` are green.

## User Interactions

Editor LSP requests over stdio:
- `textDocument/references` -> `Location[]` (or `null`).
- `textDocument/rename` -> `WorkspaceEdit` (or `null`).

## Error Handling

Every failure mode (unknown URI, no symbol under cursor, parse failure, invalid
rename target) resolves to a JSON `null` response — never a protocol error or a
panic — matching the best-effort contract of definition/hover/semantic-tokens.

## Out of Scope

- Cross-file / workspace-wide references and rename (single document only).
- Locals, function parameters, and qualified (imported) symbols.
- Rename `prepareRename` range pre-validation and references on container-element
  type positions (slice/map/pointer element types).
