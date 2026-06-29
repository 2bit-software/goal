# Verify — quality

- Tests are meaningful, not tautological: the backend test asserts BOTH cascaded
  markers are emitted AND that a direct-Node type does NOT acquire isExpr, then
  proves runtime behavior parity via a real `go build`/`go test` against a
  reference type-switch (a dropped marker would fail to COMPILE, not silently
  pass). The sema test asserts the registry cascade and exhaustiveness at both
  levels distinctly.
- Mirror discipline honored: internal/ and selfhost/ edits are line-for-line;
  the selfhost port gate compiles the .goal as Go and passes.
- Helpers avoid recursive closures (no selfhost precedent) in favor of plain
  recursion threading (seen, accumulator).
- No scope creep: parser untouched (cascade design); no foreign.go change; no
  change to existing flat-sealed paths.
- Fixpoint-safe: emitted Go byte-identical for all existing constructs.

No CRITICAL/MAJOR findings.

## Assumptions
- Map-iteration nondeterminism in the cascade does not reach emitted Go
  (SealedImpls order is sema metadata; marker order follows source-ordered
  EmbeddedIfaces) — confirmed by FIXPOINT OK.
