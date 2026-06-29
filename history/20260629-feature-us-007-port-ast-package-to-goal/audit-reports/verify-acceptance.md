# Verify — Acceptance Coverage — US-007

| Acceptance criterion | Evidence | Result |
|----------------------|----------|--------|
| selfhost/ast holds ast as goal source importing token | selfhost/ast/{ast,walk,goal_decl,goal_expr,goal_stmt}.goal; `import "goal/internal/token"` present | PASS |
| Transpiles via smoke gate + generated Go compiles | TestPortedAstPackage -> selfhost.BuildTranspiled over {token,ast} | PASS |
| Existing ast tests pass against transpiled package | TestPortedAstPackage -> selfhost.BuildAndTest(internal/ast, ../ast/ast_test.go, deps={token}) | PASS |
| Project gates green | `task check` (full `go test ./...`) + `task build` + `task fixpoint` (FIXPOINT OK) | PASS |

All acceptance criteria covered by automated tests; no CRITICAL/MAJOR gaps.

## Assumptions
- dump.go excluded by design (debug-only, unreferenced).
