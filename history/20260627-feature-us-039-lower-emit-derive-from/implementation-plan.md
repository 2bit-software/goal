# Implementation Plan — US-039 derive/from lowering

## File Inventory

### New Files
None.

### Modified Files
| File | Changes |
|------|---------|
| `internal/backend/emit.go` | Route `ast.FuncDerive` in `funcDecl` to a new `deriveDecl`; recognize the `...derive(src)` spread inside `compositeLit` only when invoked from a derive body (handled via `deriveDecl` building its own override/spread set, so `compositeLit` keeps rejecting a stray `...derive`). Add `deriveDecl`, `deriveOverrides`, `genConversion`, `resolveField` (emitter methods). |
| `internal/backend/lower.go` | Add string-level conversion helpers ported from `internal/pass/derive.go`: `derefType`, `ptrInner`, `arrElem`, `mapKV`, `elemConv`, `splitReturnType` (target+fallible from result list), and a case-insensitive `findSemaField`. |
| `internal/backend/backend_test.go` | Add `deriveCases` list, `TestASTEngineDeriveBehavioralTier` (build+vet over the 3 features/12 cases), `TestASTEngineDeriveEncoding` (pin identity / total-leaf / fallible-leaf / slice shapes + `from func` strip). |
| `prd.json` | Set US-039 `passes: true` (after green). |
| `progress.txt` | Append the US-039 entry + any new Codebase Pattern. |

## Dependency Graph

1. `lower.go` helpers (pure string functions over resolved type strings) — no deps.
2. `emit.go` `deriveDecl` + `resolveField` + `genConversion` — depend on (1) and
   `sema.Info` (Structs, FromRegistry) + emitter `gensym`/`exprText`.
3. `funcDecl` dispatch for `ast.FuncDerive` -> `deriveDecl`.
4. Tests.

## Interface Contracts

```go
// emit.go
func (e *emitter) deriveDecl(d *ast.FuncDecl)            // emits the whole conversion func
type deriveOverride struct{ Name string; Skip bool; Value ast.Expr }
func (e *emitter) deriveOverrides(body *ast.BlockStmt) (overrides []deriveOverride, hasSpread bool, ok bool)
func (e *emitter) genConversion(name, srcName, srcType, tgtType string, fallible bool, overrides []deriveOverride) (string, error)
func (e *emitter) resolveField(dst, srcExpr, sf, tf string, fallibleOK bool, errName string) ([]string, error)

// lower.go (package-level, pure)
func derefType(s string) string
func ptrInner(s string) (string, bool)
func arrElem(s string) (n, elem string, ok bool)
func mapKV(s string) (k, v string, ok bool)
func elemConv(a, b string, reg map[[2]string]sema.ConvEntry) (func(string) string, error)
func findSemaField(fields []sema.Field, name string) (sema.Field, bool)
```

## Integration Points

- `funcDecl` (emit.go) switch on `d.Mod`: add `case ast.FuncDerive: e.deriveDecl(d); return`
  before the gensym-scope setup (a derive func is emitted wholesale by deriveDecl).
- `deriveDecl` reads `e.info.Structs` and `e.info.FromRegistry` (resolved by
  `sema.Resolve`, already called in `backend.Transpile`).
- Override values are emitted via `e.exprText` (AST expr, not string parse); a
  `Field: _` (value is `Ident "_"`) is a skip.
- Temp names via `e.gensym` (one error name per fallible derive; one value temp per
  fallible field; loop index `i`/`k` for container recursion mirror the legacy text).

## Testing Strategy

- Behavioral tier (`corpus.RunCompile`, `-short`-skipped): the 3 features/12 cases
  build + vet clean through `backend.Transpile`.
- Encoding test: assert presence of `func uuidToString(` (from-strip), the identity
  assignment, the total-leaf call, the fallible `if ... != nil { return out, ... }`,
  and the slice `make(...)` + indexed loop.
- Full gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
