# Implementation Plan — US-028

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/typecheck/checker.go` | Defines `TypeChecker` interface + `GoTypesChecker` (the existing transpile→go/types crutch) implementing it via `Load` + the three depth checks. |
| `internal/typecheck/checker_test.go` | Drives the depth checks through the `TypeChecker` interface value over the existing harness fixtures; asserts behavior parity with calling the concrete functions. |

### Modified Files
| File | Changes |
|------|---------|
| `cmd/goal/main.go` | `runDepthChecks` resolves diagnostics through a `typecheck.TypeChecker` value instead of calling `Load`/`Check*` directly. |

## Design

```go
// TypeChecker is the depth-checking seam (REWRITE-ARCHITECTURE §3.2, decision 4):
// it runs the type-aware guarantees (implements, must-use, no-zero-value) over a
// goal package and returns their diagnostics. The current implementation is the
// go/types-over-lowered-Go crutch (GoTypesChecker); a native goal checker can
// replace it later WITHOUT caller changes.
type TypeChecker interface {
    Check(pkg *project.Package) ([]Diagnostic, error)
}

// GoTypesChecker transpiles the package to Go, loads it into go/types (Load), and
// runs every depth check over the typed view. It is the default TypeChecker.
type GoTypesChecker struct{}

func (GoTypesChecker) Check(pkg *project.Package) ([]Diagnostic, error) {
    p, err := Load(pkg)
    if err != nil {
        return nil, err
    }
    var diags []Diagnostic
    diags = append(diags, CheckImplements(p)...)
    diags = append(diags, CheckMustUse(p)...)
    diags = append(diags, CheckNoZeroValue(p)...)
    return diags, nil
}
```

`cmd/goal/main.go`:
```go
func runDepthChecks(pkg *project.Package) ([]typecheck.Diagnostic, error) {
    var tc typecheck.TypeChecker = typecheck.GoTypesChecker{}
    return tc.Check(pkg)
}
```

## Rationale

- `Check(pkg) ([]Diagnostic, error)` is the seam, not a `Load`-returning method,
  because the whole point (decision 4) is that a native checker — which would NOT
  produce a `*types.Package` — can satisfy the same interface. The diagnostics are
  the stable contract; `Load`/`*Package` are go/types-specific internals.
- `Load` and the `Check*` functions stay exported (their own tests still exercise
  them directly), so this is purely additive plus one caller rewrite.

## Testing

- `checker_test.go`: build a package via the existing `pkgOf` helper, run it
  through `var tc TypeChecker = GoTypesChecker{}; tc.Check(pkg)`, and assert the
  diagnostics equal the concatenation of the three `Check*` calls over `Load`'s
  `*Package` (behavior parity), plus a transpile-error case returns an error.
- All existing `internal/typecheck/*_test.go` stay green (Load + each Check*).
- Verify gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.

## Out of Scope

- Native type checker; any change to depth-check logic/messages/positions.
