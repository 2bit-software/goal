# Verification (new implementation)

- `go build ./internal/sema/ ./internal/token/` succeeds.
- sema.Analyze mirrors corpus.SemaCheck's existing parse+Resolve+Check path,
  which is already proven over the whole check corpus by sema_checker_test.go.
- sema.HasErrors / Diagnostic.Render are trivial and match the deleted
  check.HasErrors / check.Diagnostic.Render semantics (Render drops the src arg
  because Line/Col are carried on Pos).
- token.OffsetToPosition reproduces check.OffsetToPosition's clamping and 1-based
  Line/Col; it is exercised by the lsp/typecheck tests once they are repointed.

Full behavioral verification is the project verifyCommands (`task check`,
`task build`) run after cutover + cleanup.

## Assumptions
- Equivalence criterion: the migrated consumers must produce identical output to
  before (lsp ranges, typecheck findings, corpus marker matching), validated by
  the existing test suites — no new behavior is intended.
- The US-003 parity gate (corpus/parity_test.go) is retired by this story since
  the legacy checker it compares against is being deleted; documented in
  DECISIONS.md.
