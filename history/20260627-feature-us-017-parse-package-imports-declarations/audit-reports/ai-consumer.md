# Audit: AI-Consumer Readiness

The spec is implementable without guessing: the data model (ast.File, GenDecl,
FuncDecl, ImportSpec/ValueSpec/TypeSpec, type-expression nodes) is fully defined in
`internal/ast`, and the token vocabulary is fixed in `internal/token`. Acceptance
criteria map directly to test assertions over the parsed declaration list.

No CRITICAL or MAJOR findings.

## Assumptions
- `ParseFile(src) (*ast.File, error)` is the public entry point (matches the existing
  `lexer.Tokens` / `corpus`-style free-function conventions).
- The parser tokenizes once via `lexer.Tokens` rather than streaming `Next()`, for
  simpler lookahead.
