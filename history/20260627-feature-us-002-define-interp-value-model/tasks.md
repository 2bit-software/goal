# Tasks — US-002 interp value model

## Task 1: Implement the interp Value model and its tests — COMPLETED

**Files**: `internal/interp/value.go`, `internal/interp/value_test.go`

**Spec coverage**: FR-1, FR-2, FR-3, FR-4, FR-5, FR-6 (all).

**Work**:
- Create `internal/interp/value.go`:
  - Package doc referencing REWRITE-ARCHITECTURE.md §4 (universal tagged union,
    NOT the Go backend's (T,error)/*T optimizations).
  - `Kind` int discriminant: KindNil, KindInt, KindFloat, KindString, KindBool,
    KindStruct, KindSlice, KindMap, KindFunc, KindVariant; with String().
  - `Value` struct + `Variant{TypeID, Tag, Fields}`, plus StructValue/MapValue/
    FuncValue wrappers.
  - Constructors: IntVal, FloatVal, StrVal, BoolVal, NilVal, StructVal, SliceVal,
    MapVal, FuncVal, VariantVal.
  - `(Value) Field(name) (Value, bool)`, `(Value) Equal(Value) bool`,
    `(Value) String() string`.
- Create `internal/interp/value_test.go` (package interp, stdlib testing, NO
  testify):
  - Construct a Variant and each primitive/composite Value.
  - Read a Variant field back by name (present → ok, absent → !ok).
  - Assert Equal true for equal values incl. Variants, false for differing ones.
  - Assert String() is non-empty/readable for each kind.

**Verification**:
- `go build ./...`
- `go vet ./...`
- `go test ./internal/interp -count=1 -run . -v`
- `go test ./... -count=1`

**Dependencies**: none (foundation).
