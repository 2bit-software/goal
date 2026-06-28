# Plan Audit — Buildability

- Dependency order valid: fields.go (leaf) → check.go wiring → corpus test. No
  forward references.
- Interface contracts agree: `CheckFields(*ast.File, *Info) []Diagnostic` matches
  `CheckExhaustive`'s shape and `Check`'s aggregation; `Diagnostic` already
  defined in sema/check.go.
- File paths verified to exist: internal/sema/{check.go}, internal/corpus/, and
  testdata/check/08-no-zero-value/ (9 .goal files) all present.
- Reused helpers confirmed present in sema/check.go: `exprName`, `plural`,
  `pronoun`, `visitorFunc`. New helper `quoteJoin` has no collision (legacy's is
  in package check, not sema).
- Integration point specific: append in sema.Check; corpus.SemaCheck already
  forwards sema diagnostics — verified in sema_checker.go.

Each component compiles in order. No CRITICAL/MAJOR findings.

## Assumptions

- The manifest already indexes the 08-no-zero-value cases as KindCheck (confirmed:
  9 cases present under that prefix).
