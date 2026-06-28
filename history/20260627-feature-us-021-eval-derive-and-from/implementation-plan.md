# Implementation Plan — US-021 Eval derive and from

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/derive.go` | Runtime evaluation of `derive func` conversions: register derive decls, intercept calls, build the target struct Value field-by-field (identity / registry total / registry fallible / nested struct / slice / array / map), evaluate bodied overrides, refuse unsourced/unconvertible fields and pointer/Option recursion loudly. Local type-string splitters so interp never imports internal/backend. |
| `internal/interp/derive_test.go` | Unit tests over a 12-derive-convert shape (`derive_nested_struct.goal`): total derive producing the expected nested target struct (identity + registry bridge + nested struct), plus fallible success/error and unsourced-field refusal. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/interp/interp.go` | Add `derives map[string]*ast.FuncDecl` field; in `registerFuncs`, route `fn.Mod == ast.FuncDerive` into the derives map instead of the callable scope (and register a bodyless derive, which the current `fn.Body == nil` guard skips). Construct the map in `New`. |
| `internal/interp/eval.go` | In `evalCallMulti`, intercept an Ident-callee call whose name is a registered derive (not shadowed by a local binding) and route to `evalDerive` before the generic function-value path. |

## Package Structure

```
internal/interp/
  interp.go        (modified: derives map, registerFuncs routing)
  eval.go          (modified: derive call interception)
  derive.go        (new: derive evaluation + local type splitters)
  derive_test.go   (new: tests)
```

## Dependency Graph

1. `interp.go` — `ip.derives` registry + routing in registerFuncs (foundation).
2. `derive.go` — `evalDerive` + conversion helpers (reads ip.info, ip.derives).
3. `eval.go` — call interception dispatches to `evalDerive` (uses 1 + 2).
4. `derive_test.go` — exercises the wired path (uses all above).

## Interface Contracts

```go
// interp.go
type Interp struct {
    // ... existing ...
    derives map[string]*ast.FuncDecl // name -> derive func decl
}

// derive.go
// evalDerive evaluates a call to a registered derive func, returning its result
// values: a total derive yields [target]; a fallible derive yields [target, err].
func (ip *Interp) evalDerive(decl *ast.FuncDecl, call *ast.CallExpr, scope *Env) ([]Value, error)

// deriveConvert builds the target struct Value of sema type tgtType from the
// source value src of sema type srcType, applying overrides then the implicit
// same-named-field fill. Returns the value, an optional propagated conversion
// error value (fallible path), and a real interp error.
func (ip *Interp) deriveConvert(name string, src Value, srcType, tgtType string,
    fallible bool, overrides []deriveOverride, scope *Env) (Value, Value, bool, error)

// convertFieldValue converts a source field value of sema type sf to target sema
// type tf. The bool reports a propagated (fallible) conversion error in errVal.
func (ip *Interp) convertFieldValue(v Value, sf, tf string, fallible bool) (out Value, errVal Value, propagated bool, err error)

type deriveOverride struct { Name string; Value ast.Expr; Skip bool }

// Local type-string splitters (mirror backend/lower.go, no backend import):
func deriveTargetType(fl *ast.FieldList) (tgt string, fallible bool, ok bool)
func deriveOverridesOf(body *ast.BlockStmt) []deriveOverride
func findFieldFold(fields []sema.Field, name string) (sema.Field, bool)
func derefTypeName(s string) string
func sliceElem(s string) (string, bool)
func arrayElem(s string) (n, elem string, ok bool)
func mapKeyVal(s string) (k, v string, ok bool)
```

Reuses existing interp helpers: `structFields`, `zeroValue`, `baseTypeName`,
`typeExprString` is backend-only so a local `typeName(ast.Expr)` (or reuse of
existing baseTypeName over the param type) renders source/target type strings.
Registry conversions are invoked via the existing `callFunc` against a root-scope
lookup of `ConvEntry.Name` (same pattern as `callConversion`).

## Integration Points

- `New` (interp.go): initialize `derives` map; `registerFuncs` populates it.
- `evalCallMulti` (eval.go): the new interception sits alongside the existing
  Result/Option/host/method interceptions, before the generic callee path.
- `deriveConvert` reads `ip.info.Structs` (via `structFields`) and
  `ip.info.FromRegistry`; override expressions evaluate against a fresh child of
  the root scope binding the source parameter name to the source value.

## Testing Strategy

- `derive_test.go` parses a `derive_nested_struct`-shaped program through
  `internal/parser` + `internal/sema` (mirroring sibling tests like
  defaults_test.go / question_test.go), constructs `New(file, info)`, evaluates a
  call `upgrade(Person{...})` via `evalCallMulti` against the root scope, and
  asserts the produced `PersonV2` has Name (identity), Home.Street (identity), and
  Home.Zip == Code{v:"<zip>"} (registry bridge through nested struct recursion).
- Add a fallible-derive test (success + propagated error) and an unsourced-field
  refusal test. stdlib `testing` only, no testify.
- Run the full project gates: `go build ./...`, `go vet ./...`,
  `go test ./... -count=1`. Confirm `go list -deps ./internal/interp` stays free of
  internal/backend, internal/typecheck, go/types.
