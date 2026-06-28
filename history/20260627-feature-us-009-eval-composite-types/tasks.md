# Implementation Tasks — US-009 Eval composite types

## Task 1: Composite-literal, selector, and index evaluation
**Status**: completed
**Files**: internal/interp/eval.go
**Depends on**: (none — value carriers already exist)
**Spec coverage**: FR-1 (struct literal + field access), FR-2 (slice literal +
indexing), FR-3 (map literal + indexing — read side)
**Verify**: `go build ./...`

### Instructions
- Add `evalExpr` cases for `*ast.CompositeLit`, `*ast.SelectorExpr`,
  `*ast.IndexExpr`.
- `evalCompositeLit(c, scope)`: dispatch on `c.Type` —
  - `*ast.ArrayType` → slice: evaluate each non-KeyValue element in order into a
    `[]Value`; `SliceVal(elems...)`.
  - `*ast.MapType` → map: each element is a `*ast.KeyValueExpr`; evaluate key via
    `mapKeyString` (string-keyed v1) and value; build `MapVal(entries)`.
  - `*ast.Ident` (struct type name) → struct: each element is a
    `*ast.KeyValueExpr` whose Key is an `*ast.Ident` field name; build
    `StructVal(typeName, fields)`. Positional elements → descriptive error.
- `evalSelector(s, scope)`: eval `s.X`; if `KindStruct` read `Struct.Fields[Sel]`
  (absent field → error). Non-struct → descriptive error (package/variant
  selectors are later stories).
- `evalIndex(e, scope)`: eval `e.X`; `KindSlice` → eval index (must be int),
  bounds-check, return element; `KindMap` → `mapKeyString(key)`, return entry or
  a defined zero (`NilVal`) when absent. Other kinds → error.
- `mapKeyString(v)`: require `KindString`, else descriptive error (non-string
  keys deferred).
- Keep all errors `fmt.Errorf("interp: ...")` house style.

## Task 2: Range-for and index/field assignment targets
**Status**: completed
**Files**: internal/interp/interp.go
**Depends on**: Task 1
**Spec coverage**: FR-3 (key assignment), FR-4 (index/field assignment targets),
FR-5 (range-for over slices and maps)
**Verify**: `go build ./...`

### Instructions
- Add `*ast.RangeStmt` case to `execStmt` → `execRange(s, scope)`:
  - Eval `s.X`. Open a range scope (`scope.NewChild()`).
  - Slice: iterate ascending index `i`; key = `IntVal(i)`, value = element.
  - Map: iterate entries with SORTED keys (determinism); key = `StrVal(k)`,
    value = entry.
  - Bind `s.Key`/`s.Value` each iteration in a fresh child scope: `:=` (DEFINE)
    defines, `=` (ASSIGN) assigns; a `_` ident or nil is skipped. Run `s.Body`
    via `execBlock`; recover break/continue exactly like `execFor` (break stops,
    continue advances, returnSignal/error propagate).
- Refactor `bindTargets`: extract a per-target `assignTarget(lhs, v, tok, scope)`
  helper. Preserve the existing `*ast.Ident` behavior (DEFINE → Define; ASSIGN →
  Assign; compound → read/applyBinary/Assign). Add:
  - `*ast.IndexExpr` target: eval `X`; slice → bounds-checked element write
    (`X.Slice[i] = v`); map → `Map.Entries[key] = v` (insert/update). Only
    plain `=`/`:=` make sense for index targets; a compound on an index target
    reads the current element first.
  - `*ast.SelectorExpr` target: eval `X`; struct → `Struct.Fields[Sel] = v`.
  - Any other target kind → descriptive refusal (unchanged for unknown forms).
- `bindTargets` loops calling `assignTarget` per LHS/value pair.

## Task 3: Acceptance test
**Status**: completed
**Files**: internal/interp/composite_test.go (new)
**Depends on**: Task 1, Task 2
**Spec coverage**: all FRs; the AC unit test
**Verify**: `go test ./internal/interp/... -count=1`

### Instructions
- stdlib `testing` only (NO testify). Use the `newInterp` parse+resolve helper
  pattern from call_test.go (parser.ParseFile + sema.Resolve) and run a function
  via `ip.evalExpr(call("fn"), ip.root)`.
- Tests:
  - Struct literal + field access reads back assigned fields.
  - Slice literal + index reads elements; out-of-range index errors.
  - Map literal + index reads values; key assignment updates the map.
  - Range over a slice collects ascending index/element pairs (e.g. sum or
    concatenation).
  - Range over a map collects each key/value (sorted, deterministic).
  - The combined AC program builds and reads structs, slices, and maps, ranges
    over a slice and a map, and asserts the collected results.
  - Error cases: out-of-range slice index, non-string map key, field access on a
    non-struct — each a descriptive error.
- Full gate: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
