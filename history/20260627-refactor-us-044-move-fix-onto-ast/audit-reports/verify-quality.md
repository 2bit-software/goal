# Verify — Quality

- Behavior-preserving refactor: public API (File/Change/Report/Level) unchanged;
  cmd/goal and internal/lsp consume fix as before with no edits.
- The four rules read structure off the parsed tree, eliminating the token-index
  fragility (e.g. the comma-in-type-text hazards of the old scan). Comment safety
  is retained via a raw byte-range `spanHasComment` guard, since the parser drops
  comments.
- Conservative contract preserved: an unparseable source or any unmatched shape
  is a no-op; near-matches that would drop a comment or escape an Option pointer
  record a Skip rather than emit unsafe code.
- Full suite (build, vet, all package tests) green; no new dependencies.
