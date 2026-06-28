# Implementation Plan — US-009 Eval composite types

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/composite_test.go` | Unit tests for struct/slice/map literals, field/index access, key assignment, and range-for (the AC test). |

### Modified Files
| File | Changes |
|------|---------|
| `internal/interp/eval.go` | Add `evalExpr` cases for `*ast.CompositeLit`, `*ast.SelectorExpr`, `*ast.IndexExpr`; helpers `evalCompositeLit` (struct/slice/map dispatch on `Type`), `evalSelector`, `evalIndex`, and `mapKeyString`. |
| `internal/interp/interp.go` | Add `execStmt` case for `*ast.RangeStmt` -> `execRange`; extend `bindTargets` to accept `*ast.IndexExpr` and `*ast.SelectorExpr` assignment targets via a new `assignTarget` helper. |

## Package Structure

All changes are inside the existing `internal/interp` package. No new packages,
no new external dependencies (ast/token/sema + stdlib only).

```
internal/interp/
  eval.go            (modified — expression cases)
  interp.go          (modified — range stmt + assignment targets)
  composite_test.go  (new — AC tests)
```

## Dependency Graph

1. `value.go` carriers (StructValue/[]Value/MapValue) — ALREADY EXIST, no change.
2. `eval.go` composite-literal + selector + index evaluation (reads #1).
3. `interp.go` range-for and index/field assignment targets (reads #2 for the
   range subject and for reading current element/field values).
4. `composite_test.go` exercises #2 and #3.

## Interface Contracts

```go
// eval.go
func (ip *Interp) evalCompositeLit(c *ast.CompositeLit, scope *Env) (Value, error)
func (ip *Interp) evalSelector(s *ast.SelectorExpr, scope *Env) (Value, error)
func (ip *Interp) evalIndex(e *ast.IndexExpr, scope *Env) (Value, error)
func mapKeyString(v Value) (string, error) // v1 string-keyed maps

// interp.go
func (ip *Interp) execRange(s *ast.RangeStmt, scope *Env) error
func (ip *Interp) assignTarget(lhs ast.Expr, v Value, tok token.Kind, scope *Env) error
```

- `evalCompositeLit` dispatches on `c.Type`: `*ast.ArrayType` (Len==nil → slice;
  arrays treated as slices for v1), `*ast.MapType` (map, KeyValueExpr elements),
  `*ast.Ident` (struct type → keyed StructVal).
- `evalSelector`: `X` must evaluate to a struct; reads `Struct.Fields[Sel]`.
  Non-struct X / absent field → descriptive error.
- `evalIndex`: slice (int index, bounds-checked) or map (string key; absent →
  zero/Nil). Non-collection → error.
- `execRange`: slice → key=IntVal(i ascending), value=elem; map → key=StrVal,
  value, keys sorted for determinism. Honors `:=` (Define) / `=` (Assign) and
  the blank identifier `_`. Recovers break/continue like execFor.
- `assignTarget` factors the per-target binding: `*ast.Ident` (existing
  Define/Assign/compound), `*ast.IndexExpr` (slice element / map key), and
  `*ast.SelectorExpr` (struct field). `bindTargets` calls it per target.

## Integration Points

- `internal/interp/eval.go` `evalExpr` switch — three new cases.
- `internal/interp/interp.go` `execStmt` switch — one new case (`*ast.RangeStmt`).
- `internal/interp/interp.go` `bindTargets` — route each LHS through
  `assignTarget` so non-Ident targets (index/field) are supported. The
  short-var `:=`, plain `=`, and compound paths for `*ast.Ident` are preserved
  exactly.

## Testing Strategy

`internal/interp/composite_test.go`, stdlib `testing` only (NO testify), parses
real goal source via `parser.ParseFile` + `sema.Resolve` (the `newInterp`
helper pattern from call_test.go), runs a function, and asserts returned values
— plus a couple of direct-AST tests for error cases (out-of-range index,
non-string key). Mirrors the table-driven style of eval_test.go / control_test.go.

## Requirement Coverage

- FR-1 struct literal + field access → evalCompositeLit (struct) + evalSelector.
- FR-2 slice literal + indexing → evalCompositeLit (slice) + evalIndex.
- FR-3 map literal + indexing + key assignment → evalCompositeLit (map) +
  evalIndex + assignTarget (IndexExpr/map).
- FR-4 index/field assignment → assignTarget.
- FR-5 range-for → execRange.
