# Verify — Acceptance (US-050)

All acceptance criteria pass against the implemented code.

| Criterion | Result |
|-----------|--------|
| references returns all sites for func/type/variant via AST graph | PASS — TestReferencesFunction / TestReferencesTypeName / TestReferencesEnumVariant |
| includeDeclaration toggle | PASS — TestReferencesIncludeDeclarationToggle (with == without+1) |
| rename returns WorkspaceEdit with edit at every reference (AC test) | PASS — TestRenameProducesEditsAtEveryReference (edits == count of occurrences; applying them replaces every occurrence) |
| variant/type distinctness | PASS — TestReferencesEnumVariant asserts no Green/type leak |
| null fallbacks (unknown URI / no symbol / unparseable / invalid name) | PASS — TestRenameInvalidName, TestReferencesNoSymbolAndUnparseable, TestReferenceRenameHandlersNullForUnknownURI |
| capability advertisement | PASS — TestServerAdvertisesReferencesAndRename |

## Verify commands (prd.json verifyCommands)

- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./... -count=1` — all packages ok (including internal/lsp)

No CRITICAL or MAJOR findings. Recommend pass.
