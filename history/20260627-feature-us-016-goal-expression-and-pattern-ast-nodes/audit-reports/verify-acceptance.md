# Verify: Acceptance Coverage — US-016

All prd.json verifyCommands green: `go build ./...`, `go vet ./...`,
`go test ./... -count=1` (all packages ok).

| Acceptance criterion | Evidence | Status |
|----------------------|----------|--------|
| AST defines MatchExpr, MatchArm, VariantPattern, RestPattern, UnwrapExpr, VariantLit, LabeledArg, SpreadElement | `internal/ast/goal_expr.go` declares all eight; `go build` compiles | PASS |
| Construction VariantLit and destructuring VariantPattern are distinct node types | `TestWalkGoalExprChildren` asserts `%T` of each differs and equals `*ast.VariantLit` / `*ast.VariantPattern` | PASS |
| Both walk correctly (Walk descends into each node's children once) | `TestWalkGoalExprChildren` uses collector + assertChildren over MatchExpr (with both patterns), VariantLit, UnwrapExpr, SpreadElement | PASS |
| Build, vet, test green | full suite run above | PASS |

No acceptance criterion is uncovered.
