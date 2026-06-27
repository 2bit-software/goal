# Implementation Plan — US-034 Lower and emit Result and Option

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| (none) | The encoders fold into the existing `internal/backend` package, mirroring US-033's pattern (no separate `lower` pkg). |

### Modified Files
| File | Changes |
|------|---------|
| `internal/backend/lower.go` | Add gensym name constants (`okName`/`errName`/`valName`/`someName`/`optBase`) and small AST helpers: `resultOptionKind(*ast.FuncType)` (classify a result type as open-E Result / Option / none + return success-type expr), `isErrorIdent(ast.Expr)`, `matchQualifier(*ast.MatchExpr)` (first arm's enum qualifier name), `usesIdent(ast.Node, string) bool`. |
| `internal/backend/emit.go` | Thread current-function kind on the emitter (`fnKind` field, save/restore in `funcDecl`); lower the Result/Option result signature in `funcSig`; lower `return Result.Ok/Err` and `return Option.Some/None` in the `ReturnStmt` case (function-scoped); lower `Option[T]` -> `*T` in the `IndexExpr` expr case; intercept a statement-position `*ast.MatchExpr` in the `ExprStmt` stmt case and dispatch to Result/Option match lowering; add an ident-rename map (`renames`) consulted in the `Ident` expr case. |
| `internal/backend/backend_test.go` | Add a behavioral-tier test (`TestASTEngineResultOptionBehavioralTier`) driving every 03-result and 04-option transpile case from `corpus/manifest.json` through `backend.Transpile` + `corpus.RunCompile`; `-short`-skipped (spawns the go toolchain). Optionally a focused emit/format assertion for one Result and one Option case. |
| `prd.json` | Set US-034 `passes: true` (after green). |
| `progress.txt` | Append the US-034 entry; add any reusable pattern to the top block. |

## Package Structure

```
internal/backend/
  backend.go        (unchanged — Transpile already calls sema.Resolve)
  lower.go          (+ Result/Option encoders & helpers)
  emit.go           (+ fnKind state, signature/return/match/Option lowering, renames)
  backend_test.go   (+ Result/Option behavioral-tier test)
```

## Dependency Graph

1. `lower.go` helpers + name constants (no new deps; uses `ast`, `sema`, `strings`, `unicode` already imported).
2. `emit.go` wiring (depends on 1).
3. `backend_test.go` test (depends on 2; uses `internal/corpus` from the external `backend_test` package, as existing tests do).

## Interface Contracts

```go
// lower.go
const (
    okName   = "__goal_ok"
    errName  = "__goal_err"
    valName  = "__goal_v"
    someName = "__goal_some"
    optBase  = "__goal_o"
)

type roKind int
const (
    roNone roKind = iota
    roResultOpen // Result[T, error]
    roOption     // Option[T]
)

// resultOptionKind classifies a single unnamed result type; success is the T expr.
func resultOptionKind(t *ast.FuncType) (kind roKind, success ast.Expr)

func isErrorIdent(x ast.Expr) bool          // x is *ast.Ident{Name:"error"}
func matchQualifier(m *ast.MatchExpr) string // first VariantPattern enum name, "" if none
func usesIdent(n ast.Node, name string) bool // ast.Walk for an *ast.Ident with that name
```

```go
// emit.go — emitter gains:
type emitter struct {
    // ...existing...
    fnKind  roKind            // enclosing function's Result/Option kind
    renames map[string]string // active match-arm binding renames (Ident -> gensym)
}
```

Emission shapes (mirror `internal/pass/{result,option}.go`):

- Result signature: `(__goal_ok <T>, __goal_err error)`.
- `return Result.Ok(X)` -> `return X, nil`; `return Result.Err(X)` -> `return __goal_ok, X`.
- Result statement match: `<lhs>, __goal_err := <scrut>` then `if __goal_err != nil { <errBody> } else { <okBody> }`; Ok binding renamed to `__goal_v` (lhs `_` if unused), Err binding renamed to `__goal_err`.
- Option type: `*<T>`.
- `return Option.None` -> `return nil`; `return Option.Some(x)` -> `return &x` (identifier) else `__goal_some := x` + `return &__goal_some`.
- Option statement match: `if __goal_o := <scrut>; __goal_o != nil {` `[<bind> := *__goal_o]` `<someBody>` `} else {` `<noneBody>` `}`.

## Integration Points

- `internal/backend/emit.go` `funcDecl` / `funcSig` — signature lowering.
- `internal/backend/emit.go` `stmt` `*ast.ReturnStmt` case — constructor lowering (guarded by `e.fnKind`).
- `internal/backend/emit.go` `stmt` `*ast.ExprStmt` case — statement-match interception.
- `internal/backend/emit.go` `expr` `*ast.IndexExpr` case — `Option[T]` -> `*T`.
- `internal/backend/emit.go` `expr` `*ast.Ident` case — apply `renames`.
- `sema.Info.FuncSignatures` (already populated by `sema.Resolve` in `backend.Transpile`) — used to refuse closed-E Result matches.

## Testing Strategy

- External `backend_test` package (already used) imports `internal/corpus`.
- `TestASTEngineResultOptionBehavioralTier`: load `corpus/manifest.json`, filter
  `KindTranspile` + `ModeFile` cases whose `Input` has prefix
  `features/03-result/examples/` or `features/04-option/examples/`, run each through
  `corpus.RunCompile(repoRoot, c, corpus.TranspilerFunc(backend.Transpile))`;
  `t.Fatalf` on zero cases; `-short`-skipped.
- Reuse the repoRoot/manifest helpers already present in `backend_test.go`
  (US-033 added the same shape for 01-enums/07-implements).
