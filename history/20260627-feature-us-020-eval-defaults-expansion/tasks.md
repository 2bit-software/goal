# Implementation Tasks — US-020 Eval defaults expansion

## Task 1: Expand `...defaults` in evalCompositeLit + zeroValue helper
**Status**: completed
**Files**: `internal/interp/eval.go`
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2, FR-3, FR-4
**Verify**: `go build ./...` && `go vet ./...`

### Instructions
- In `internal/interp/eval.go`, add a method
  `func (ip *Interp) zeroValue(typ string, depth int) Value` mirroring backend
  `zeroLit`:
  - Trim the type. Prefix checks first: `*`, `[]`, `map[`, `chan`, `func`,
    `interface` (non-empty), and the bare names `any`/`error` -> `NilVal()`,
    EXCEPT `[]T` which returns `SliceVal()` (empty slice) so range/len stay valid.
  - `string` -> `StrVal("")`; `bool` -> `BoolVal(false)`; integer kinds
    (int,int8..int64,uint..uint64,uintptr,byte,rune) -> `IntVal(0)`;
    float32/float64/complex64/complex128 -> `FloatVal(0)`.
  - Otherwise, a named type: if `depth < 8` and `ip.info.Structs[base]` exists
    (base = type name stripped of `*` and `pkg.`), return a recursively
    zero-filled `StructVal(base, {field: zeroValue(field.Type, depth+1)})`.
  - Fallback for an unknown named type -> `NilVal()`.
- In `evalCompositeLit`'s `case *ast.Ident:` branch, replace the strict
  KeyValueExpr-only loop:
  - For each elt: a `*ast.KeyValueExpr` sets `fields[name]` exactly as today; a
    `*ast.SpreadElement` with `X` an `*ast.Ident` named `defaults` sets a local
    `wantDefaults` flag; any other `*ast.SpreadElement` is a located refusal
    `interp: <pos>: unsupported spread element ... (only ...defaults)`; any other
    element keeps the existing "requires keyed field: value" refusal.
  - After the loop, if `wantDefaults`: look up `ip.info.Structs[t.Name]`; if
    absent -> located refusal `interp: cannot expand ...defaults: unknown struct
    type <name>`; else for each declared field NOT already in `fields`, set
    `fields[f.Name] = ip.zeroValue(f.Type, 0)`.
  - Return `StructVal(t.Name, fields)` unchanged.
- Reuse a small local `baseType` normalization (strip leading `*` and any
  `pkg.` qualifier) — do NOT import internal/backend or internal/scan.

## Task 2: Tests for defaults expansion
**Status**: completed
**Files**: `internal/interp/defaults_test.go` (new)
**Depends on**: Task 1
**Spec coverage**: all acceptance criteria
**Verify**: `go test ./internal/interp/... -run Defaults -count=1`

### Instructions
- New file `internal/interp/defaults_test.go`, `package interp`, stdlib
  `testing` only (no testify). Model on composite_test.go / enum_test.go using
  the `newInterp` + `evalFn` / `evalFnErr` helpers.
- Cases:
  - Primitives (08 defaults_primitives shape): a func returning
    `User{name:"x", role:RoleMember, ...defaults}` -> assert email=="",
    active==false, logins==0, name preserved. Return the User struct and read
    fields off `got.Struct.Fields`.
  - Explicit-after-defaults ordering: `...defaults` first, an explicit field
    after -> the explicit field is preserved (not zeroed).
  - Nested struct + slice defaults: a struct with an omitted named-struct field
    and an omitted slice field -> zeroed inner struct and empty slice.
  - `...derive` refusal: hand-build a CompositeLit with a SpreadElement whose X
    is Ident "derive" (or a tiny program if it parses) and assert a descriptive
    error via evalFnErr / direct evalExpr.

## Final verification (whole-project gates from prd.json)
- `go build ./...`
- `go vet ./...`
- `go test ./... -count=1`
