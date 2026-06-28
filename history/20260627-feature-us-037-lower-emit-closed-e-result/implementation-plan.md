# Implementation Plan — US-037 Lower and emit closed-E Result

## File Inventory

### New Files
None. The lowering folds into the existing internal/backend (per the project
pattern: encoders in lower.go, dispatch in emit.go — no separate `lower` package).

### Modified Files
| File | Changes |
|------|---------|
| `internal/backend/lower.go` | Add `roResultClosed` to the roKind enum; add the `resultPrelude` constant (mirrors internal/pass.ResultPreamble); add `needsResultPrelude(info)`. |
| `internal/backend/emit.go` | Emit the prelude once in `file()`; add `closedT/closedE` emitter fields; in `funcDecl` set fnKind=roResultClosed + closedT/closedE from the sema sig and allow `ast.FuncFrom` to emit as a plain func; `returnStmt` lowers closed ctors via new `emitClosedResultReturn`; `unwrap` dispatches roResultClosed -> new `unwrapClosed`; `resultMatch` routes ModeResultClosed -> new `closedResultMatch`. |
| `internal/backend/backend_test.go` | Add `errorEClosedCases` (the 3 06-error-e inputs); `TestASTEngineClosedResultBehavioralTier` (build+vet via corpus.RunCompile, -short-skipped); `TestASTEngineClosedResultEncoding` (pin prelude / Ok-Err sum / closed `?` / From-conversion shapes). |

## Package Structure

internal/backend/ (unchanged layout)
- backend.go   — Backend/Formatter/Transpile (untouched)
- emit.go      — AST -> Go text (modified)
- lower.go     — goal-construct encoders + sema accessors (modified)
- backend_test.go — external package backend_test (modified)

## Dependency Graph

1. lower.go: `roResultClosed` const, `resultPrelude`, `needsResultPrelude` (no deps).
2. emit.go: prelude emission in `file()` (depends on 1).
3. emit.go: funcDecl closed-E setup + FuncFrom passthrough (depends on 1).
4. emit.go: `emitClosedResultReturn`, `unwrapClosed`, `closedResultMatch`
   (depend on 3's emitter fields closedT/closedE; reuse existing calleeSig/
   calleeMode/armBody/gensym/bindingName/usesIdent).
5. backend_test.go tests (depend on all above).

## Interface Contracts

```go
// lower.go
const roResultClosed roKind = iota+... // after roOption
const resultPrelude = `type Result[T, E any] interface{ isResult() }
type Ok[T, E any] struct{ Value T }
type Err[T, E any] struct{ Value E }

func (Ok[T, E]) isResult()  {}
func (Err[T, E]) isResult() {}`
func needsResultPrelude(info *sema.Info) bool

// emit.go (emitter methods)
func (e *emitter) emitClosedResultReturn(x ast.Expr) bool   // return Result.Ok/Err -> Ok/Err[T,E]{Value: …}
func (e *emitter) unwrapClosed(name string, u *ast.UnwrapExpr, discard bool)
func (e *emitter) closedResultMatch(m *ast.MatchExpr)
// emitter gains fields: closedT, closedE string
```

## Integration Points

- `file()` (emit.go): after the package clause, emit `resultPrelude` once before
  the first non-import decl, guarded by `needsResultPrelude(e.info)`.
- `funcDecl()` (emit.go): compute kind; if the sema sig for d.Name is
  ModeResultClosed, set kind=roResultClosed, closedT/closedE; save+restore them
  with the existing fnKind/okName/errName/taken block. Allow ast.FuncFrom in the
  modifier guard (emit as ordinary func).
- `returnStmt()` (emit.go): `case roResultClosed: if e.emitClosedResultReturn(...)`.
- `unwrap()` (emit.go): `case roResultClosed: e.unwrapClosed(...)`.
- `resultMatch()` (emit.go): replace the `closed-E … later story` fail with a
  route to `closedResultMatch` when `e.calleeMode(m.Subject) == sema.ModeResultClosed`.

## Testing Strategy

- Behavioral tier: corpus.Case{Kind:KindTranspile, Mode:ModeFile, Input:<06 path>}
  through corpus.RunCompile(repoRoot, c, corpus.TranspilerFunc(backend.Transpile)),
  -short-skipped (spawns the go toolchain), one subtest per input.
- Encoding: backend.Transpile(mustRead(...)) and assert substrings:
  prelude (`type Ok[T, E any] struct{ Value T }`), sum ctor
  (`Ok[Config, ParseError]{Value:`), closed match type-switch
  (`case Ok[Config, ParseError]:`), From-conversion (`toApp(`), panicking default.
- Project gates: go build ./..., go vet ./..., go test ./... -count=1.
