# Verification — US-021 Parse match and patterns

## Verify gates (prd.json verifyCommands)
- `go build ./...` — pass
- `go vet ./...` — pass
- `go test ./... -count=1` — pass (all packages ok)

## Acceptance criteria → evidence
- Statement-position match parses with expected arms →
  `TestParseMatchStatementPosition` (status_match.goal): ExprStmt wrapping a
  3-arm MatchExpr; variants Pending/Active(a)/Cancelled(c) with bindings.
- Value-position match parses with expected arms →
  `TestParseMatchValuePositionVar` (status_var.goal, var initializer) and
  `TestParseMatchValuePositionReturn` (status_return.goal, return result):
  3-arm MatchExpr located in value position.
- Binding pattern records enum/variant/binding → asserted in the
  statement-position test (Active binds `a`, Cancelled binds `c`).
- Rest pattern `_` distinct node → `TestParseMatchRestPattern`
  (status_rest.goal): last arm is *ast.RestPattern.

All four match tests pass; full suite green. Conclusion: PASS.
