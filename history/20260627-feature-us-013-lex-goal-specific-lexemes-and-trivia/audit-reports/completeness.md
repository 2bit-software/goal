# Audit — Completeness

## Findings

### MINOR — `..` (two dots) behavior unspecified
The spec covers `.` (PERIOD) and `...` (ELLIPSIS) but not the intermediate
`..`. goal has no `..` lexeme; the natural outcome is two PERIOD tokens. Not a
blocker — the existing single-PERIOD path already yields this and no goal
construct uses `..`. Recommend the implementation simply not match `..` as a
unit.

### MINOR — DOC_COMMENT at end-of-file (no trailing newline)
FR-4 says `///` consumes "the rest of its line". The line-comment scanner
already terminates at EOF as well as `\n`, so a `///` with no trailing newline
is covered. Worth a test assertion but not a spec gap.

### None CRITICAL / MAJOR
All five functional requirements map to existing token kinds and a single,
local lexer seam (`scanOperator` + comment path). Happy paths and the
distinctness cases (FAT_ARROW vs ASSIGN+GTR, ELLIPSIS vs PERIODs, DOC_COMMENT
vs COMMENT) are all covered by explicit acceptance criteria.

## Assumptions

- `..` (two consecutive dots not followed by a digit) lexes as two PERIOD
  tokens; goal defines no `..` operator.
- "Retained as trivia" for this lexer-only story means "emitted as a distinct
  DOC_COMMENT token", with parser-layer attachment deferred (matches FR scope
  and the Out of Scope section).
