# Completeness Audit — US-029

## Findings

No CRITICAL or MAJOR findings. The spec mirrors an already-shipped, well-specified
guarantee (`internal/check/exhaustive.go`) with an existing, exhaustive test corpus
(`testdata/check/02-match`, 7 cases covering: clean exhaustive, non-exhaustive in
statement / return / inferred positions, rest-arm opt-out, deferred unknown enum,
Result-match skip). Behavior is pinned by inline `// want` markers, so each acceptance
criterion is independently verifiable.

- MINOR: "exhaustiveness-related case in testdata/check" is scoped to the `02-match`
  directory (the exhaustiveness feature). Resolved by selecting check cases whose input
  path is under `testdata/check/02-match`.

## Assumptions

- The sema checker for US-029 runs ONLY the exhaustiveness check (other checks land in
  US-030/031), so it is exercised against the 02-match cases — running it over other
  feature dirs is out of scope and could surface unclaimed diagnostics unrelated to
  exhaustiveness.
- Diagnostic message wording matches the legacy check verbatim so existing `// want`
  markers continue to match.
- Per the loop constraint, work stays on branch `ralph/ast-frontend-rewrite`; no branch
  is created by the workflow start step.
