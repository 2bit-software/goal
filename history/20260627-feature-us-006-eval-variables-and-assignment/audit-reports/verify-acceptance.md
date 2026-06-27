# Verify — Acceptance Coverage (US-006)

All project verify gates pass: `go build ./...`, `go vet ./...`,
`go test ./... -count=1` (every package ok).

| Acceptance criterion | Test evidence | Result |
|---|---|---|
| declare (var ± init, `:=`, const), reassign `=`, compound-assign → expected finals | `TestDeclareReassignCompound` (a=11, b=1, c=4, d=10) | PASS |
| read a declared variable returns latest value | `TestShortVarAndReadInExpression` (y = x*x = 9) | PASS |
| plain `=` updates existing binding, not a shadow | `TestAssignUpdatesExistingBindingNotShadow` (parent n→42, no child shadow) | PASS |
| var without initializer zeroes | `TestVarWithoutInitializerZeroes` (int/float/string/bool) | PASS |
| assign to undeclared name → descriptive error | `TestAssignUndeclaredErrors` (NotFoundError "missing") | PASS |
| read undefined name → descriptive error | `TestReadUndefinedErrors` (NotFoundError "nope") | PASS |

Every acceptance criterion maps to a test that asserts the required behavior.

## Assumptions
- Final values are read back from the run scope (`ip.root.NewChild()` running
  `main.Body` via `execBlock`), the same direct-eval approach US-005's test
  uses — no new public API was added for testability.
