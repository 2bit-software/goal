# Technical Requirements / Research — US-010

## Existing seam

- `evalCallMulti` (internal/interp/eval.go) is the single call seam: it resolves
  the callee to a function value and dispatches to `callFunc`. Builtins and
  method calls must be intercepted here BEFORE the generic function-value path.
- Control-flow signals ride the `(… error)` channel as sentinel error types
  (`returnSignal`, `breakSignal`, `continueSignal`) recovered via `errors.As`.
  `panic` follows the same pattern: a new `panicSignal{value Value}` that
  propagates past loop/switch/call boundaries (none of them recover it) up to
  the Go test boundary — that is the "recovered panic".
- `StructValue` is pointer-backed (`Value.Struct *StructValue`), so a
  pointer-receiver method mutating a field is visible through the caller's
  binding for free. A value-receiver method must operate on a shallow COPY of
  the struct so its mutations do not leak (Go value semantics).
- Method receiver: `FuncDecl.Recv` is a `*ast.FieldList` with one `Field`;
  `Names[0]` is the receiver name; `Type` is `*ast.Ident` (value receiver) or
  `*ast.StarExpr{X:*ast.Ident}` (pointer receiver). Methods close over the root
  (package) scope, like plain functions.

## Plan

- Build a method registry in `New` keyed `typeName -> methodName -> *FuncDecl`.
- Intercept builtin idents (`len`/`append`/`make`/`panic`) and method
  selector calls in `evalCallMulti`; fall through to the generic path otherwise.
- `make` reads a TYPE expr arg (`*ast.MapType` / `*ast.ArrayType`), reusing
  `zeroValue` for element zeros.
