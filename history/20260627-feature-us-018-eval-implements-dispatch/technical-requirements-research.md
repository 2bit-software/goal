# Technical Requirements / Research — US-018

## Existing seam

US-010 added the method registry and dispatch to internal/interp:

- `registerMethods()` (interp.go) indexes every method declaration by
  star-stripped receiver type name -> method name -> *ast.FuncDecl, so value
  (`(s T)`) and pointer (`(s *T)`) receivers register under the same `T`.
- `tryMethodCall` (eval.go) evaluates the selector receiver; when it is a
  `KindStruct` value whose `TypeID` has a method `M`, it dispatches via
  `callMethod`. Pointer receivers share the caller's `*StructValue` (mutations
  visible); value receivers bind a shallow copy.

## Key insight

Because the interpreter erases static types at runtime (REWRITE-ARCHITECTURE
§3.2) and dispatches structurally on the concrete value's `TypeID`, an
interface-typed binding simply holds the concrete struct value. Calling an
interface method therefore already dispatches to the correct concrete method —
for differently-typed values, value receivers, pointer receivers, and values
read out of a heterogeneous interface-element slice. The US-010 learning
explicitly predicted this: "US-018 (implements dispatch) builds on this
registry; an interface value dispatched to a concrete method reuses callMethod."

## Deliverable

The story's deliverable is the conformance proof: a unit test over a
07-implements shape (`internal/interp/implements_test.go`) exercising:

- differently-typed value-receiver dispatch through an interface-typed parameter,
- pointer-receiver dispatch through an interface-typed parameter (mutation
  observed), and
- dispatch over a heterogeneous slice of interface values.

No new production code is expected unless a gap surfaces during implementation;
if one does, close it minimally in eval.go/interp.go following the existing
dispatch seam.

## Verify gates (prd.json verifyCommands)

- `go build ./...`
- `go vet ./...`
- `go test ./... -count=1`

Plus the new `internal/interp` implements test. Tests use stdlib `testing`
only (no testify).
