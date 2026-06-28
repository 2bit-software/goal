# Audit — AI-Consumer Readiness

## Findings

### None CRITICAL / MAJOR
An implementing agent has everything needed:
- Target token kinds are named and already exist in `internal/token`
  (QUESTION, FAT_ARROW, ELLIPSIS, DOC_COMMENT, COMMENT, IDENT).
- The seam is identified: `internal/lexer` `scanOperator` and the comment path.
- Each acceptance criterion is phrased as an observable token-stream assertion
  ("`?` → QUESTION, EOF"), directly translatable to a table-test row.

### MINOR — Token count phrasing
Criteria say "a single token"; the emitted stream always also ends in EOF.
Tests should assert `len == 2` (lexeme + EOF) or index token[0]. This is a
test-authoring detail, already noted in the technical research.

## Assumptions

- Longest-match ordering is required: `///` before `//`, `=>` before `=`/`==`,
  `...` before `.`. This is standard scanner behavior and implied by FR-1..FR-4
  but stated here explicitly so the implementer does not regress shorter forms.
- DOC_COMMENT Lit retains the full `/// ...` text verbatim (consistent with how
  COMMENT retains its `// ...` text today).
