# Implementation Plan — US-007 Eval functions and calls

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/call_test.go` | Unit tests: recursive factorial + fibonacci (AC), multi-return divmod into `q, r :=`, arg-count mismatch error, non-function call error, undefined call error. `package interp`, stdlib testing only. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/interp/value.go` | Extend `FuncValue` to carry `Decl *ast.FuncDecl` and `Env *Env` (the defining scope) alongside `Name`. Add a `FuncValVal` (or extend `FuncVal`) constructor that takes the decl + env. Keep `FuncVal(name)` working (name-only carrier still used by value_test.go) — add a new constructor rather than break the old one. `value.go` will need to import `goal/internal/ast`. |
| `internal/interp/interp.go` | (1) In `New`, register every top-level plain `*ast.FuncDecl` (Recv==nil, Name!=nil) into `root` as a function value (so recursion + forward refs + tests resolve). (2) `Run` becomes: ensure funcs registered, look up `main`, call it with no args. (3) Add `execStmt` cases for `*ast.ReturnStmt` and `*ast.IfStmt`. (4) Return must unwind through `execBlock`/`execStmt` via a typed sentinel `returnSignal` (carries `[]Value`); `execBlock` propagates it; the call boundary intercepts it. |
| `internal/interp/eval.go` | Add `*ast.CallExpr` case to `evalExpr` (single-value position: require exactly 1 result). Add `evalCallMulti(call, scope) ([]Value, error)` doing the actual call: resolve callee to a func value, evaluate args, arity-check, bind params in a fresh child of the func's defining env, run the body, collect returns. `execAssign` (interp.go) gains the `a, b := f()` multi-result branch by calling `evalCallMulti` when `len(Lhs) > len(Rhs) == 1` and the sole RHS is a `*ast.CallExpr`. |

## Package Structure

```
internal/interp/
  value.go        (modified: FuncValue carries Decl+Env)
  env.go          (unchanged)
  eval.go         (modified: CallExpr eval + evalCallMulti)
  interp.go       (modified: func registration, Run, return/if, execAssign multi)
  call_test.go    (new)
  ...existing tests unchanged
```

## Dependency Graph

1. `value.go` — extend `FuncValue` + constructor (foundation; no deps).
2. `interp.go` — `returnSignal` type + register funcs in `New` (depends on 1).
3. `eval.go` — `evalCallMulti` + `CallExpr` case (depends on 1, 2).
4. `interp.go` — `execStmt` ReturnStmt/IfStmt + `execAssign` multi-result branch
   + `Run` calls main (depends on 2, 3).
5. `call_test.go` — tests (depends on all).

## Control-flow mechanics

- `returnSignal` is an `error` sentinel: `type returnSignal struct { vals []Value }`
  with `Error() string`. `execBlock` returns it up unchanged; `evalCallMulti`
  recovers it via `errors.As`, extracts `vals`, and treats it as a normal return
  (NOT an error). Any other error propagates as a real error.
- `if`: evaluate `Cond` (must be KindBool); run `Body` (a new child scope) when
  true, else `Else` (`*ast.BlockStmt` or `*ast.IfStmt`). `Init` stmt, if present,
  runs first in the if's own scope. A returnSignal from a branch propagates.
- A function returning no declared results yields zero values; a `return` with no
  results and fall-off-the-end both produce `returnSignal{vals: nil}`. A call in
  value position with 0 results -> descriptive error.

## Multi-return model (resolves the audit CRITICAL)

- `evalCallMulti` returns `[]Value` — the canonical N-result form. No tuple Kind
  is added to `Value`; ambiguity with a real slice is avoided by never boxing
  multiple results into one Value.
- `evalExpr`'s `CallExpr` case calls `evalCallMulti`; exactly 1 result -> that
  Value; 0 or >1 -> descriptive "multi-value call in single-value context" error.
- `execAssign`: when `len(Rhs)==1 && len(Lhs) > 1` and `Rhs[0]` is `*ast.CallExpr`,
  call `evalCallMulti`, require `len(results)==len(Lhs)`, bind positionally
  (`:=` Define / `=` Assign). Existing equal-length path unchanged.

## Reuse / constraints

- Reuse `Env.NewChild`/`Define`/`Lookup`, `applyBinary`, `zeroValue`.
- Keep internal/interp deps to errors/fmt/strconv + goal/internal/{ast,sema,token}
  (US-022 gate). No backend/typecheck/go-types.
- Verify: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
