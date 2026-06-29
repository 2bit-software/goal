# Implementation Plan — US-002

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/selfhost/selfhost.go` | Reusable harness (`package selfhost`): list of in-scope packages, read a dir's non-test `*.go` as a `project.Package`, transpile a set of packages and `go build` them in a throwaway temp module. The verification harness every later port story reuses. |
| `internal/selfhost/selfhost_test.go` | `package selfhost_test`: positive gate (all 8 covered packages transpile + build) and negative gate (a deliberately non-compiling package makes the build fail). |

### Modified Files
| File | Changes |
|------|---------|
| `internal/sema/check.go` | Rename local identifier `enum` (a goal reserved word) to `enumDecl` in `checkOneMatch` and `missingVariants` so the package is valid goal. Behavior-preserving. |
| `internal/backend/emit.go` | Rename local identifier `enum` to `enumIdent` in `variantLit` and the variant-name switch. Behavior-preserving. |

## Package Structure

```
internal/
  selfhost/
    selfhost.go        # package selfhost — harness
    selfhost_test.go   # package selfhost_test — the gate
```

`internal/selfhost` imports `goal/internal/backend`, `goal/internal/parser`,
`goal/internal/pipeline`, `goal/internal/project`. None of those import
`internal/selfhost`, so there is no cycle.

## Dependency Graph

1. Rename `enum` identifiers in `sema/check.go` and `backend/emit.go` (makes those
   packages transpile cleanly — prerequisite for a green gate).
2. `internal/selfhost/selfhost.go` (harness; depends only on existing packages).
3. `internal/selfhost/selfhost_test.go` (the gate; depends on 1 and 2).

## Interface Contracts

```go
package selfhost

// InScope lists the compiler packages the smoke gate covers, by directory name
// under internal/.
var InScope = []string{"token", "lexer", "ast", "parser", "sema", "project", "pipeline", "backend"}

// ReadPackage reads the non-_test *.go files in dir as goal source and returns a
// project.Package whose Dir is dir (so the front-end's import resolver finds the
// enclosing go.mod) and whose Name is the package clause.
func ReadPackage(dir string) (*project.Package, error)

// BuildTranspiled transpiles each package via backend.TranspilePackage, writes the
// generated Go into a throwaway temp module (module `goal`, go 1.26) under the
// module-relative directory given as the map key, and runs `go build ./...`.
// It returns a descriptive error on any transpile failure, invalid generated Go,
// or build failure; nil on success.
func BuildTranspiled(layout map[string]*project.Package) error
```

Test mirrors `internal/<pkg>` for each `InScope` entry: read from `../<pkg>`
(test cwd is `internal/selfhost`), key the layout by `internal/<pkg>`.

## Integration Points

- `backend.TranspilePackage(*project.Package) (pipeline.PackageOutput, error)` — the
  front-end driver (internal/backend/package.go).
- `project.Package` / `project.File{Path, Name, Src}` (internal/project).
- `parser.ParseFile` to read the package clause for `Name`.
- Build pattern mirrors `internal/corpus/package_runner.go` (`go/format` validity
  check optional; the `go build ./...` over a temp module is the core).

## Testing Strategy

- `TestInScopePackagesTranspileAndBuild`: build the layout for all 8 `InScope`
  packages; assert `BuildTranspiled` returns nil. This is the gate (FR-1..FR-3).
- `TestGateFailsOnNonCompilingTranspile`: build a layout with one in-memory package
  whose source transpiles but does not compile (e.g. an `int`-returning func with a
  bare `return`); assert `BuildTranspiled` returns a non-nil error (FR-3 red path).
- stdlib `testing` only (zero-dependency house rule). Runs under `task check`.
