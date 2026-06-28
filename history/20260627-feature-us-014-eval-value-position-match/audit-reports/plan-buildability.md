# Plan Audit — Buildability

- Dependency order is valid: helpers (1) -> evalMatch (2) -> evalExpr dispatch (3)
  -> tests (4). No forward references.
- Interface contracts are concrete Go signatures consistent with existing code
  (`Value`, `*Env`, `*ast.MatchExpr`, `*ast.MatchArm`, `*ast.VariantPattern` all
  exist and are already used by `execMatch`).
- File paths verified: `internal/interp/eval.go` and `internal/interp/interp.go`
  exist; `value_match_test.go` is new and does not collide (existing match test is
  `match_test.go`).
- Integration point is specific: a new `case *ast.MatchExpr` in the `evalExpr`
  switch in eval.go; no caller changes required.
- Each step compiles in order: the refactor keeps `execMatch` compiling, then the
  new functions are added, then dispatched.

No CRITICAL/MAJOR findings.

## Assumptions

- A value-position arm body is an `ast.Expr`; a statement/block body in value
  position is a descriptive refusal (consistent with the parser's arm dispatch).
- The US-022 dependency envelope is preserved — no new imports of go/types,
  internal/backend, or internal/typecheck (verified post-implementation via
  `go list -deps`).
