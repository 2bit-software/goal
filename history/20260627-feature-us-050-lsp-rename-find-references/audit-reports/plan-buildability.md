# Plan Audit — Buildability (US-050)

The plan compiles in dependency order with no forward references.

- **Order valid**: protocol types -> occurrence collector -> handlers -> routing
  -> tests. Each layer only uses lower layers plus already-existing helpers.
- **Contracts agree**: `WorkspaceEdit`/`TextDocumentEdit`/`TextEdit`/
  `versionedTextDocumentIdentifier` already exist in protocol.go (used by
  codeaction.go) — verified, no signature drift. `buildDeclIndex`,
  `offsetForPosition`, `rangeOf`, `identRange`, `named`, `name` all exist in the
  `lsp` package (definition.go/hover.go/symbols.go) and are reusable directly.
- **File paths verified**: `internal/lsp/references.go` and `references_test.go`
  do not yet exist (no conflict). `s.buffer(uri)` returns (text, version, ok) —
  matches rename's version-pinning need.
- **Integration points specific**: server.go `handle` switch adds two `case`
  arms routing to `s.reply(m.ID, s.references(...))` / `s.rename(...)`; initialize
  `ServerCapabilities` literal gains two bool fields.

## MINOR

- The occurrence visitor duplicates definition.go's structural keying. Acceptable
  — it records a different payload (symKey vs target Range). Keeping it a separate
  visitor avoids entangling the two handlers; consistent with how hover.go has its
  own `hoverVisitor`.

No CRITICAL/MAJOR findings. Cleared to tasks.
