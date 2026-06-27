# Implementation Plan — US-009

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `testdata/package/cross-file-demo/math.goal` | Cross-file fixture: enum match + closed-E Result (from pipeline_package_test.go `mathGoal`). |
| `testdata/package/cross-file-demo/types.goal` | Sibling decls (Shape, MathErr enums) (from `typesGoal`). |
| `testdata/package/cross-file-demo/pkg.json` | Descriptor: `{name:"demo", imports:{}}`. |
| `testdata/package/foreign-derive/conv.goal` | Foreign-derive fixture: `derive func make(o *ext.Outer) Local` (from foreign_test.go `src`). |
| `testdata/package/foreign-derive/pkg.json` | Descriptor: `{name:"conv", imports:{"goal/internal/pipeline/testdata/extpkg":"internal/pipeline/testdata/extpkg"}}`. |
| `internal/corpus/package_runner.go` | `PackageTranspiler` seam + `PackageTranspilerFunc` adapter + `RunPackage`. |
| `internal/corpus/package_runner_test.go` | Drives every `Mode=package` case through `pipeline.TranspilePackage`. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/corpus/corpus.go` | Add `PackageSpec` type + `Package *PackageSpec` field on `Case`. |
| `internal/corpus/generate.go` | Discover `testdata/package/*/pkg.json`, glob `.goal` files, emit `Mode=package` cases. |
| `internal/corpus/generate_test.go` | Count file-mode vs package-mode transpile; exempt package cases from the Expected-non-empty shape check. |
| `corpus/manifest.json` | Regenerated to include the 2 package cases. |

## Package Structure

```
testdata/package/
  cross-file-demo/{math.goal,types.goal,pkg.json}
  foreign-derive/{conv.goal,pkg.json}   # imports the existing internal/pipeline/testdata/extpkg
internal/corpus/
  corpus.go            # + PackageSpec, Case.Package
  generate.go          # + package-fixture discovery
  package_runner.go    # + PackageTranspiler, RunPackage
  package_runner_test.go
```

## Dependency Graph

1. `corpus.go` model extension (`PackageSpec`, `Case.Package`) — no deps.
2. On-disk fixtures + `pkg.json` descriptors — no deps.
3. `generate.go` package discovery — depends on 1, 2.
4. Regenerated `corpus/manifest.json` — depends on 3.
5. `package_runner.go` (`RunPackage`) — depends on 1.
6. Tests — depend on 3, 4, 5.

## Interface Contracts

```go
// corpus.go
type PackageSpec struct {
    Name    string            `json:"name"`
    Files   []string          `json:"files"`             // repo-relative .goal paths
    Imports map[string]string `json:"imports,omitempty"` // import path -> repo-relative foreign dir
}
type Case struct { /* …existing… */ Package *PackageSpec `json:"package,omitempty"` }

// package_runner.go
type PackageTranspiler interface {
    TranspilePackage(pkg *project.Package) (pipeline.PackageOutput, error)
}
type PackageTranspilerFunc func(*project.Package) (pipeline.PackageOutput, error)
func (f PackageTranspilerFunc) TranspilePackage(p *project.Package) (pipeline.PackageOutput, error)

func RunPackage(root string, c Case, pt PackageTranspiler) error
```

`RunPackage`: builds `*project.Package{Dir: root/Input, Name, Files}`, transpiles,
asserts every `Files`/`Tests` entry is valid Go (`format.Source`), then writes the
package + each declared foreign import (under its module-relative tail) into a
temp `module goal` and runs `go build ./...`.

## Integration Points

- `pipeline.TranspilePackage` — package transpile seam (already exists).
- `analyze.DefaultResolver` (invoked inside TranspilePackage from `pkg.Dir`) —
  resolves the in-module foreign import for the foreign-derive case.
- `cmd/corpus-gen` — regenerates `corpus/manifest.json` (already wired via
  `go generate ./internal/corpus`).

## Testing Strategy

- `package_runner_test.go` (internal `package corpus`): load `corpus/manifest.json`,
  run every `Mode=package` case through `PackageTranspilerFunc(pipeline.TranspilePackage)`
  via `RunPackage`; loud `t.Fatalf` on zero package cases; `-short`-skip the compile.
- `generate_test.go`: assert file-mode transpile == 51, package-mode == 2,
  check == 50, doctest == 4; exempt package cases from the Expected check.
