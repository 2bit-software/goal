# Technical Plan — US-010 Eval builtins and methods

## Files

- `internal/interp/interp.go`
  - Add `panicSignal{value Value}` (error sentinel, like `returnSignal`), with
    `Error()` rendering the panic value. It is NOT recovered by execFor /
    execSwitch / callFunc, so it propagates to `Run` and out — the Go test
    boundary observes it (the "recovered panic").
  - Add a method registry `methods map[string]map[string]*ast.FuncDecl` on
    `Interp`, populated by a new `registerMethods()` called from `New`
    (alongside `registerFuncs`). Keyed by receiver type name (star-stripped) ->
    method name -> declaration.
- `internal/interp/eval.go`
  - `evalCallMulti`: intercept, in order, BEFORE the generic callee path —
    1. an `*ast.Ident` callee whose name is a builtin AND is not shadowed by a
       scope binding -> `evalBuiltin`;
    2. an `*ast.SelectorExpr` callee that resolves to a struct method ->
       `callMethod`; otherwise fall through to the existing function-value path
       (so a struct field holding a func, or a package call deferred to US-011,
       still behaves as today).
  - `evalBuiltin(name, call, scope)`:
    - `len`: slice length / string byte length / map entry count -> IntVal.
    - `append`: first arg slice + each remaining arg -> a NEW slice value.
    - `make`: arg[0] is a TYPE expr — `*ast.MapType` -> empty map;
      `*ast.ArrayType` -> slice of `n` element-zero values (n = arg[1], default
      0), reusing `zeroValue`.
    - `panic`: evaluate the operand and return `panicSignal{value}`.
  - `callMethod(decl, recv, args)`: bind the receiver in a fresh child of root;
    a value receiver gets a shallow struct COPY (`copyStructValue`) so its
    mutations don't leak; a pointer receiver shares the pointer-backed struct.
    Bind params, run the body, recover `returnSignal` (mirrors `callFunc`).
  - Helpers: `isBuiltin`, `recvTypeName`, `recvName`, `recvIsPointer`,
    `copyStructValue`.

## Tests

`internal/interp/builtins_test.go` (package interp, stdlib testing):
- `len`/`append` on a slice; `make` of a map (write + read back); `make` of a
  slice; a recovered panic (errors.As to `panicSignal`, value asserted); a
  value-receiver method (no leak) and a pointer-receiver method mutating a
  field visible to the caller.

## Dependency discipline

No new imports beyond ast/token/errors/fmt already used; interp stays free of
go/types, internal/backend, internal/typecheck (US-022 gate).
