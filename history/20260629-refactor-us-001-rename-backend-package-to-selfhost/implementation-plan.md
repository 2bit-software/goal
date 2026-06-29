# Implementation Plan — US-001

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `selfhost/backend/arity.go.goal` etc. (6 `.goal` files) | Verbatim goal copies of the 6 non-test `internal/backend/*.go` files |
| `internal/backend/backend_selfhost_test.go` | The 12 fixture-free tests split out of backend_test.go so the behavioral gate can copy a corpus-free, compilable suite |

Naming: the ported files are `selfhost/backend/<base>.goal` (e.g. `emit.goal`),
matching the prior ports (ReadPackage/Discover name generated Go off `<base>.go`).

### Modified Files
| File | Changes |
|------|---------|
| `internal/backend/backend_test.go` | Remove the 12 self-contained tests (moved to the new file); keep fixture/corpus tests + helpers (mustRead/readFixture) and adjust imports if any become unused |
| `internal/selfhost/port_test.go` | Add `TestPortedBackendPackage` (compile + behavioral gates) using `discoverPorted` |
| `prd.json` | Set US-001 `passes: true` |
| `progress.txt` | Append the iteration entry |

## Steps

1. Copy each `internal/backend/<f>.go` (arity, backend, doctest, emit, lower,
   package) to `selfhost/backend/<f>.goal` verbatim. No edits (zero reserved-word
   collisions confirmed).
2. Split backend_test.go: move the 12 self-contained tests into
   `backend_selfhost_test.go` (package backend_test), give it exactly the imports
   those tests use; remove them from backend_test.go and drop any now-unused
   import there. Verify both files compile via `task check`.
3. Add `TestPortedBackendPackage` to internal/selfhost/port_test.go:
   - `discoverPorted` for token, lexer, ast, parser, sema, project, pipeline, backend.
   - COMPILE: `BuildTranspiled` over the 8-entry layout.
   - BEHAVIORAL: `BuildAndTest("internal/backend", backendPkg, [the 12 split test
     files], deps={token,lexer,ast,parser,sema,project,pipeline})`.
4. Run verify commands: `task check`, `task build`, `task fixpoint`, plus
   `go test ./internal/selfhost -run TestPortedBackendPackage`.

## Reuse
- `selfhost.BuildTranspiled`, `selfhost.BuildAndTest`, `discoverPorted` (existing).
- Pattern mirrors TestPortedSemaPackage / TestPortedPipelinePackage exactly.

## Risk / rollback
- If a transpile/build defect surfaces, it is a real front-end bug to fix here
  (the gates are the point). Rollback is removing selfhost/backend + the new test.
