# Implementation Plan — US-005 Port token package to goal

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `selfhost/token/token.goal` | The token package as goal source — verbatim copy of `internal/token/token.go` (Go superset is valid goal). Declares `package token`. |
| `internal/selfhost/port_test.go` | Verification: transpiles `selfhost/token` via the US-002 harness (compiles), then runs the existing `internal/token` tests against the transpiled package. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/selfhost/selfhost.go` | Add a reusable `BuildAndTest(relDir, pkg, testFiles)` helper that transpiles a ported package into a temp `module goal`, copies the named existing test files alongside the generated Go, and runs `go test` on it. Reused by US-006+. |

## Package Structure

```
selfhost/
  main.goal            (existing skeleton, package main)
  token/
    token.goal         (new, package token)
internal/selfhost/
  selfhost.go          (+ BuildAndTest helper)
  selfhost_test.go     (existing smoke gate)
  port_test.go         (new, ported-token verification)
```

## Dependency Graph

1. `selfhost/token/token.goal` — leaf, no internal deps.
2. `internal/selfhost/selfhost.go` BuildAndTest helper — depends on existing
   ReadPackage/writePackage patterns only.
3. `internal/selfhost/port_test.go` — depends on (1) and (2) plus the existing
   `internal/token/token_test.go` it copies in.

## Interface Contracts

```go
// BuildAndTest transpiles pkg into a throwaway `module goal` temp module at
// relDir (e.g. "internal/token"), copies each path in testFiles into that same
// directory (so same-package white-box tests compile against the transpiled
// source), and runs `go test ./<relDir>`. Returns a descriptive error on
// transpile failure, invalid Go, or a test failure; nil when the tests pass.
func BuildAndTest(relDir string, pkg *project.Package, testFiles []string) error
```

## Integration Points

- `internal/selfhost/port_test.go` reads `selfhost/token` with
  `project.Discover("../../selfhost/token")` (test cwd is internal/selfhost),
  asserts a single `token` package, runs it through `selfhost.BuildTranspiled`
  (criterion 2: transpiles + compiles), then `selfhost.BuildAndTest` with the
  existing `../token/token_test.go` (criterion 3: existing tests pass).
- No changes to `internal/token`; it remains the trusted reference.
- `selfhost/token` is auto-discovered by the US-004 bootstrap/fixpoint targets;
  both stages emit it identically (fixpoint stays byte-identical) and the
  selfhost main does not import it (`go build ./selfhost` unaffected).

## Testing Strategy

- Reuse stdlib `testing` (zero-dependency project; no testify).
- `port_test.go` in package `selfhost_test`, mirroring the existing
  `selfhost_test.go` layout.
- Verification is two-pronged: BuildTranspiled (compiles) and BuildAndTest
  (existing tests pass), so a regression in either the transpile or behavior
  fails the gate.
- Project gates: `task check` (vet + `go test ./...`) and `task build`.
