# Implementation Plan — US-018 Eval implements dispatch

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/implements_test.go` | Conformance test (US-018): interface method dispatch over a 07-implements shape — differently-typed value-receiver dispatch through an interface parameter, pointer-receiver dispatch through an interface parameter (mutation observed), and dispatch over a heterogeneous slice of interface values. |

### Modified Files
| File | Changes |
|------|---------|
| (none expected) | Dispatch already rides on the US-010 method registry (`registerMethods` / `tryMethodCall` / `callMethod`). Exploratory tests confirm value-receiver, pointer-receiver-via-interface-param, and heterogeneous-slice dispatch all work. Production code is touched only if a gap surfaces while writing the test. |

## Package Structure

```
internal/interp/
  interp.go            (existing — registerMethods, callMethod; unchanged)
  eval.go              (existing — tryMethodCall, evalCallMulti; unchanged)
  implements_test.go   (NEW — US-018 conformance test)
```

## Dependency Graph

1. Existing dispatch seam (US-010): `registerMethods`, `tryMethodCall`,
   `callMethod` — already present.
2. `internal/interp/implements_test.go` — depends on (1) and on the existing
   test helpers `newInterp` / `evalFn` (call_test.go / composite_test.go).

## Interface Contracts

No new production signatures. The test exercises the existing seam:

```go
// existing, reused:
func (ip *Interp) tryMethodCall(sel *ast.SelectorExpr, call *ast.CallExpr, scope *Env) ([]Value, bool, error)
func newInterp(t *testing.T, src string) *Interp        // call_test.go
func evalFn(t *testing.T, ip *Interp, name string) Value // composite_test.go
```

## Integration Points

The test drives whole goal programs through `newInterp` (parser.ParseFile +
sema.Resolve + New) and evaluates top-level functions via `evalFn(t, ip, name)`,
asserting returned `Value` Kind/Str/Int. No CLI or backend wiring.

## Testing Strategy

`internal/interp/implements_test.go`, package `interp`, stdlib `testing` only
(no testify). Cases:

- `TestImplementsValueReceiverDifferentTypes` — `Stringer{ Describe() string }`
  with two struct implementers (`Point`, `Label`); a `render(s Stringer) string`
  function returns `s.Describe()`; assert each concrete `Describe` runs.
- `TestImplementsPointerReceiverThroughInterface` — `Resetter{ Reset() }` with
  a pointer-receiver `Counter.Reset`; call through an interface parameter and
  assert the mutation is observed (no `&` needed).
- `TestImplementsHeterogeneousSliceDispatch` — `Shape{ Area() int }` with two
  implementers in a `[]Shape{...}`, ranged and summed, asserting each concrete
  `Area` ran.
- `TestImplementsMissingMethodIsLoud` — calling an unimplemented interface
  method surfaces a descriptive error (loud, not silent).

Verify gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
