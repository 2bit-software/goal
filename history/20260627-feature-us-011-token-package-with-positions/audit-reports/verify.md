# Verification — US-011 token package with positions

## Verify gates (prd.json verifyCommands)
- `go build ./...` → OK
- `go vet ./...` → OK
- `go test ./... -count=1` → all packages ok, including `goal/internal/token`

## Acceptance criteria
1. "internal/token defines Kind constants for every goal lexeme and
   Pos{Offset,Line,Col}" → MET. `internal/token/token.go` defines `Kind` with
   constants for Go tokens, the literal classes, every operator/delimiter, the
   reserved keywords, the goal-specific `QUESTION`/`FAT_ARROW`/`ELLIPSIS`/
   `DOC_COMMENT`, and `ILLEGAL`/`EOF`; `Pos{Offset,Line,Col}` is defined.
2. "A unit test asserts Kind/String round-trips and Pos ordering" → MET.
   `TestKindStringRoundTrip` round-trips every operator and keyword kind through
   `String()`→`Lookup`; `TestPosOrdering` asserts offset ordering (irreflexive,
   asymmetric). Supporting tests: goal-specific lexemes (`=>` one token, not `=`+`>`),
   contextual keywords are IDENT, unknown lookups return `(ILLEGAL,false)`, class
   predicates, out-of-range String, Pos.String.

## Findings
None (no CRITICAL/MAJOR/MINOR). Feature works exactly as specified.

## Test run
8 tests in `goal/internal/token`, all PASS. Full suite green.
