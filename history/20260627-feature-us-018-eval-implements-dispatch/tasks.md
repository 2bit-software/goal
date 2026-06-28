# Implementation Tasks — US-018 Eval implements dispatch

## Task 1: Add interface-dispatch conformance test
**Status**: completed
**Files**: `internal/interp/implements_test.go` (new)
**Depends on**: (none — rides on the existing US-010 dispatch seam)
**Spec coverage**: FR-1, FR-2, FR-3, FR-4 + acceptance criteria
**Verify**: `go test ./internal/interp -run TestImplements -count=1`, then full gates `go build ./...`, `go vet ./...`, `go test ./... -count=1`

### Instructions
Create `internal/interp/implements_test.go` (package `interp`, stdlib `testing`
only — no testify). Reuse the existing helpers `newInterp` (call_test.go) and
`evalFn` (composite_test.go); do NOT redeclare them. Add:

- `TestImplementsValueReceiverDifferentTypes` — a 07-implements shape
  `type Stringer interface { Describe() string }` with two struct implementers
  `Point` and `Label` (each `... struct implements Stringer { ... }`), each with
  a value-receiver `Describe()`; a `render(s Stringer) string` returns
  `s.Describe()`; assert `render(Point{...}) == "point"` and
  `render(Label{...}) == "label"` (differently-typed dispatch, each concrete
  impl runs).
- `TestImplementsPointerReceiverThroughInterface` — `type Resetter interface
  { Reset() }`, `type Counter struct implements Resetter { n int }` with
  pointer-receiver `func (c *Counter) Reset() { c.n = 0 }`; a
  `use(r Resetter) {...}` calls `r.Reset()`; a driver builds a `Counter{n:5}`,
  passes it to `use`, and returns `c.n`; assert the result is 0 (pointer-receiver
  mutation observed through the interface, no `&`).
- `TestImplementsHeterogeneousSliceDispatch` — `type Shape interface { Area()
  int }`, two implementers `Square`/`Rect`; build `[]Shape{Square{...},
  Rect{...}}`, range and sum `.Area()`; assert the total proves each concrete
  `Area` ran.
- `TestImplementsMissingMethodIsLoud` — call an interface method the concrete
  value's type does not declare and assert a descriptive (non-nil) error.

Each interface uses a single method or all-returning methods to avoid the
pre-existing parser limitation (a void interface method followed by another
method fails to parse). Drive programs through `newInterp`; evaluate via
`evalFn` / `ip.evalExpr(call(name), ip.root)`. Assert on `Value` Kind/Str/Int.

If (and only if) a genuine dispatch gap surfaces, fix it minimally in
`internal/interp/eval.go` or `interp.go` along the existing
`tryMethodCall`/`callMethod` seam, then re-run the verify gates.
