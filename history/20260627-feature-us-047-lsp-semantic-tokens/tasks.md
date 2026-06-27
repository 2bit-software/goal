# Tasks — US-047 LSP semantic tokens

## T1 — Wire protocol types + legend (foundation)
- Files: internal/lsp/protocol.go
- Add `SemanticTokensOptions`, `SemanticTokensLegend`, `SemanticTokensParams`,
  `SemanticTokens`; extend `ServerCapabilities` with `SemanticTokensProvider`.
- Add legend index constants + ordered `semanticTokenTypes`/`semanticTokenModifiers`
  slices + `defaultSemanticLegend()`.
- Spec coverage: FR-1, FR-4.
- Depends on: none.

## T2 — Compute + classify tokens from the AST
- Files: internal/lsp/semantictokens.go (new)
- `computeSemanticTokens`, `classifyToken`, `astRoles` (roleVisitor over ast.Walk),
  delta-encoder.
- Spec coverage: FR-2, FR-3, FR-4.
- Depends on: T1.

## T3 — Handler + capability advertisement
- Files: internal/lsp/server.go
- `semanticTokens` handler; route `textDocument/semanticTokens/full`; advertise
  `SemanticTokensProvider` in the initialize reply.
- Spec coverage: FR-1, FR-2.
- Depends on: T1, T2.

## T4 — Tests
- Files: internal/lsp/semantictokens_test.go (new), internal/lsp/server_test.go (edit)
- TestComputeSemanticTokensEnumMatchQuestion (the AC sample), well-formedness,
  empty/unparseable, handler, and capability-advertised assertions.
- Spec coverage: all ACs.
- Depends on: T1, T2, T3.

## Status
- T1 — completed
- T2 — completed
- T3 — completed
- T4 — completed

## Coverage check
- FR-1: T1, T3. FR-2: T2, T3. FR-3: T2. FR-4: T1, T2.
- Files: protocol.go (T1), semantictokens.go (T2), server.go (T3), tests (T4).
