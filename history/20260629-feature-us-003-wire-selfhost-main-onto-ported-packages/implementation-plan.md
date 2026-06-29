# Implementation Plan — US-003

## File Inventory

### New Files
None (a transient nested `go.mod` is written into the gitignored `_bootstrap/`
tree at bootstrap time, not committed).

### Modified Files
| File | Changes |
|------|---------|
| `selfhost/main.goal` | Imports `goal/internal/{backend,pipeline,project}` → `goal/selfhost/{backend,pipeline,project}`; update doc comment. |
| `selfhost/**/*.goal` (token, lexer, ast, parser, sema, project, pipeline, backend, typecheck) | Every `goal/internal/X` import → `goal/selfhost/X`. |
| `Taskfile.yml` | `bootstrap`: write a nested `go.mod` (`module goal`, `go 1.26`) into each emit dir (`_bootstrap/s1`, `_bootstrap/s2`) and build each stage with `go build -C _bootstrap/sN -o ../../bin/goal-c-N ./selfhost`. |
| `internal/selfhost/selfhost.go` | `BuildAndTest`: rewrite `goal/internal/` → `goal/selfhost/` in copied white-box test files so their sibling imports match the relocated (selfhost) package-under-test. |
| `internal/selfhost/port_test.go` | All `BuildTranspiled` layout keys, `BuildAndTest` `relDir`, and `deps` keys: `internal/X` → `selfhost/X`. |

## Dependency Graph

1. Rewrite selfhost `.goal` imports (FR-1) — no dependency.
2. `BuildAndTest` test-file import rewrite — needed before port_test keys flip.
3. `port_test.go` key flip — depends on 1 + 2.
4. `Taskfile.yml` bootstrap nested-module build — depends on 1.

## Interface Contracts

- `BuildAndTest(relDir string, pkg *project.Package, testFiles []string, deps map[string]*project.Package) error` — signature unchanged; internally rewrites `goal/internal/` → `goal/selfhost/` in copied test file bytes before writing them into the temp module.
- Taskfile `bootstrap` produces `bin/goal-c-1`, `bin/goal-c-2` built from `_bootstrap/sN/selfhost` (nested `module goal`).

## Integration Points

- `selfhost/main.goal` `run()` → `project.Discover` + `backend.TranspilePackage` now resolve to `goal/selfhost/*`.
- `Taskfile.yml` `fixpoint` (deps: [bootstrap]) is unchanged; it diffs goal-c-1 vs goal-c-2 emit of `./selfhost`, now a genuine differential proof.

## Testing Strategy

- `task check` — `go vet ./...` + `go test ./...` (includes `internal/selfhost` port gates with flipped keys, and corpus transpile/behavioral/check tiers via the trusted compiler).
- `task build` — both binaries build.
- `task fixpoint` — bootstrap (3-stage, nested module) + `diff -r` byte-identical.
- AC grep: `grep -rn 'goal/internal/' selfhost/` returns nothing (imports/comments all flipped).
