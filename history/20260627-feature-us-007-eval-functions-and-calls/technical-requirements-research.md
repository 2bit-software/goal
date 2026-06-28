# Technical Requirements & Research — US-007

## Existing seams (internal/interp)

- `interp.go`: `execStmt` is the statement-dispatch seam (ExprStmt, DeclStmt,
  AssignStmt, EmptyStmt today). `Run` finds `func main` and runs its body in a
  child scope. `findMain` walks `file.Decls` for `*ast.FuncDecl`.
- `eval.go`: `evalExpr` dispatches expression nodes; `*ast.CallExpr` currently
  falls through to the "unsupported expression" refusal.
- `env.go`: `Env` with Define/Lookup/Assign and parent-linked child scopes.
- `value.go`: `FuncValue{Name}` is a name-only carrier today; it must grow to
  carry the declaration so a call can bind params and run the body. `Value`
  already supports multi-value via separate return; the interpreter needs a
  return-signal mechanism.

## Approach

- Register every top-level `*ast.FuncDecl` (plain funcs, no receiver) into the
  root Env as function values BEFORE running main, so recursion and forward
  references resolve through normal `scope.Lookup`.
- Extend `FuncValue` to carry the `*ast.FuncDecl` (the closure over the
  top-level/root scope). Keep `Name` for rendering.
- Add `*ast.CallExpr` evaluation: resolve the callee (Ident -> Lookup) to a
  func value, evaluate args, bind to a fresh child of the function's defining
  scope, run the body, collect the return value(s).
- Multiple return values: model a return as a non-local control signal carrying
  `[]Value`. Use a sentinel error type (e.g. `returnSignal`) threaded through
  execStmt/execBlock — the standard tree-walker pattern — so a `return` deep in
  nested blocks unwinds to the call boundary. A call yields a single Value when
  one result, and a tuple Value (slice) when multiple. For US-007 the test uses
  single-return factorial/fibonacci; multi-return is exercised by an additional
  test (e.g. divmod) per the AC "supports multiple return values".
- factorial/fibonacci need `if`/`else` + `return`. Implement `*ast.IfStmt` and
  `*ast.ReturnStmt` here (the minimum recursion needs). The full control-flow
  suite (for, switch, break/continue) is US-008.

## Constraints

- Zero dependency, stdlib `testing` only (no testify).
- internal/interp must keep depending only on errors/fmt/strconv +
  goal/internal/{ast,sema,token} (US-022 dependency gate); do not import
  backend/typecheck/go-types.
- Verify gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
