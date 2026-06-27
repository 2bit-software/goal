# Completeness Audit — US-046

## Findings

- MINOR: FR-2 says ranges are "faithful" but the exact byte span (keyword-start
  vs name-start) is not specified. Resolved by deferring to the existing
  symbols_test.go contract (selection range ⊇ name; full range ⊇ declaration;
  bodyless decls do not overrun the next). No ambiguity that blocks work.
- MINOR: "the AST front-end's lexer" for range widening is an implementation
  choice; the user-visible behavior (token-tight range, line-end fallback) is
  pinned by diagnostics_test.go and is preserved.
- None CRITICAL or MAJOR. The spec's acceptance criteria map 1:1 to existing,
  passing tests, which fully constrain happy path, empty state, and malformed
  input.

## Edge cases covered

- Empty source -> non-nil empty outline (TestCollectSymbolsEmptyAndPartial).
- Malformed/partial source -> no panic (same test).
- Unknown document URI -> empty outline (TestDocumentSymbolHandler).
- Bodyless decl adjacency -> no range overrun (TestCollectSymbolsBodylessDoesNotSwallow).

## Assumptions

- The existing LSP test suite is the authoritative behavioral contract; no new
  observable behavior is introduced.
- Migrating the checker itself off analyze is out of scope (LSP package path
  needs analyze foreign resolution).
