# Verify Audit: Acceptance — US-023

Each spec acceptance criterion maps to a passing test in
`internal/parser/goal_stmt_test.go`:

- from_storage (from func + bodyless derive func) → TestParseFromDerive. PASS.
- slice (bodyless derive func toIDs) → TestParseFromDerive. PASS.
- to_storage (bodied from func + bodied derive func) → TestParseFromDerive. PASS.
- bank (bare assert) → TestParseAssertBare. PASS.
- message (printf-message assert, 1 arg) → TestParseAssertMessage. PASS.
- multiple (bare + top-level-comma split with call-internal commas + `%` bare)
  → TestParseAssertTopLevelCommaSplit. PASS.
- add (1 doctest, input/expected spot-check) → TestParseDoctests. PASS.
- multi (2 doctests) → TestParseDoctests. PASS.
- mixed (half=0 doctests, double=1) → TestParseDoctests. PASS.
- Walk visits new nodes once, no skip/dup → TestWalkNewNodes. PASS.
- go build/vet/test ./... → all green.

No CRITICAL/MAJOR findings.

## Assumptions
- Doc comments attach to the immediately-following function only; a doc run
  before a non-function declaration is treated as prose and dropped.
