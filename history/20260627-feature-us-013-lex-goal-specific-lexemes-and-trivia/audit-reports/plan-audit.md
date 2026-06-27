# Plan Audit — US-013

## Findings

### None CRITICAL / MAJOR
The plan traces every functional requirement (FR-1..FR-5) to a concrete change
in `internal/lexer/lexer.go` plus test rows in `lexer_test.go`. File paths are
verified to exist. No new files, no new dependencies, no public API change. The
dependency graph is a valid order (token → lexer → tests) with no cycles.

### MINOR — `peek2` lookahead
The plan introduces a two-rune lookahead for `///` and `...`. The existing
lexer only has `peek()`. Implementer should add a minimal `peek2()` (or decode
the rune after `rdOffset`) — trivial and self-contained. Flagged so it is not
overlooked.

### MINOR — package doc comment
The plan correctly notes the `internal/lexer` package doc still says these
lexemes are "deferred to US-013"; updating it avoids stale docs. Cosmetic.

## Assumptions
- `...` is matched only as exactly three dots; `..` falls through to two PERIOD
  tokens (goal has no `..`).
- Tests live in the existing internal `package lexer` test file (no corpus
  dependency, so no external-test-package requirement applies here).
