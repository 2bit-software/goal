# Implementation Tasks — US-050 LSP rename and find references

## Task 1: Add protocol types
**Status**: pending
**Files**: internal/lsp/protocol.go
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2, FR-3, FR-5
**Verify**: `go build ./internal/lsp/`

### Instructions
- Add `ReferenceContext{IncludeDeclaration bool json:"includeDeclaration"}`.
- Add `ReferenceParams{TextDocument textDocumentIdentifier, Position Position,
  Context ReferenceContext}`.
- Add `RenameParams{TextDocument textDocumentIdentifier, Position Position,
  NewName string json:"newName"}`.
- Add `ReferencesProvider bool json:"referencesProvider,omitempty"` and
  `RenameProvider bool json:"renameProvider,omitempty"` to `ServerCapabilities`.
- `WorkspaceEdit`/`TextDocumentEdit`/`TextEdit`/`versionedTextDocumentIdentifier`
  already exist — reuse them.

## Task 2: Occurrence collector + handlers
**Status**: pending
**Files**: internal/lsp/references.go
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-2, FR-3, FR-4, error handling
**Verify**: `go build ./internal/lsp/`

### Instructions
- Define `symKey{kind symKind, enum, name string}` with kinds func/type/variant.
- `collectOccurrences(src, file, idx declIndex) []occurrence` where
  `occurrence{start,end int, key symKey, isDecl bool}`: seed declaration-name
  occurrences (isDecl=true) for FuncDecl/EnumDecl(+variants)/SealedInterfaceDecl/
  GenDecl TypeSpec, then `ast.Walk` an `occVisitor` mirroring definition.go's
  `refVisitor` structural keying (CallExpr.Fun ident + method Sel, SelectorExpr
  over a known enum, VariantLit/VariantPattern, Field/PayloadField/ValueSpec/
  TypeSpec/ImplementsClause type idents), recording references with isDecl=false.
- `resolveOccurrences(src, line, char) (key symKey, occ []occurrence, ok bool)`:
  offsetForPosition -> parser.ParseFile -> buildDeclIndex -> collectOccurrences;
  find the occurrence covering the offset; return all same-key occurrences.
- `references(raw) []Location`: decode ReferenceParams, buffer lookup, resolve,
  map same-key occurrences to Location (rangeOf); drop isDecl when
  !IncludeDeclaration; return nil on any failure (JSON null).
- `rename(raw) *WorkspaceEdit`: decode RenameParams, validate NewName via
  `isIdent` (letter/`_` then letter/digit/`_`), buffer lookup (text+version),
  resolve, build one TextEdit per same-key occurrence, wrap in WorkspaceEdit
  with a versioned TextDocumentEdit; nil on any failure.
- Best-effort: never panic; nil result marshals to JSON null.

## Task 3: Server routing + capabilities
**Status**: pending
**Files**: internal/lsp/server.go
**Depends on**: Task 1, Task 2
**Spec coverage**: FR-5
**Verify**: `go build ./internal/lsp/`

### Instructions
- In initialize `ServerCapabilities`, set `ReferencesProvider: true` and
  `RenameProvider: true`.
- In `handle` switch, add `case "textDocument/references": s.reply(m.ID,
  s.references(m.Params))` and `case "textDocument/rename": s.reply(m.ID,
  s.rename(m.Params))`.

## Task 4: Tests
**Status**: pending
**Files**: internal/lsp/references_test.go
**Depends on**: Task 1-3
**Spec coverage**: all ACs
**Verify**: `go test ./internal/lsp/ -count=1`

### Instructions
- Reuse `offsetOfNth`/`cursorAt` from definition_test.go (same package).
- TestReferencesFunction: all call sites + decl of a function.
- TestReferencesIncludeDeclarationToggle: false omits decl, true includes it.
- TestReferencesEnumVariant / TestReferencesTypeName.
- TestRenameProducesEditsAtEveryReference: assert a TextEdit at every occurrence
  (the required AC test).
- TestRenameInvalidName / TestReferencesNoSymbol / Unparseable -> null.
- TestReferenceRenameHandlers (open URI vs unknown) + TestServerAdvertises
  references/rename providers.

## Task 5: Verify gates
**Status**: pending
**Files**: (none)
**Depends on**: Task 1-4
**Spec coverage**: AC build/vet/test
**Verify**: `go build ./...` && `go vet ./...` && `go test ./... -count=1`
