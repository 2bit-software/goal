# Plan Audit — Coverage

Every spec requirement maps to a plan element:

- FR-1 (true is a no-op) -> `execAssert` returns nil when `cond.Bool`.
- FR-2 (false panics, located, with message) -> `execAssert` returns
  `panicSignal{StrVal(located "assertion failed: <cond>")}`.
- FR-3 (printf-message form) -> message branch using `goArgs` + `fmt.Sprintf`.
- FR-4 (non-bool refusal) -> `cond.Kind != KindBool` descriptive error.
- AC test -> `assert_test.go` cases (true no-op, false panic, message form,
  non-bool).

No scope creep: only an execStmt case + one new file (execAssert + exprText) +
tests. `exprText` is the one piece not 1:1 with a requirement, but it is required
to satisfy FR-2's "includes the asserted condition".

No CRITICAL/MAJOR findings.

## Assumptions

- The condition text comes from an AST renderer (`exprText`) rather than source
  bytes, since the interp holds no source (US-022 native-only envelope).
