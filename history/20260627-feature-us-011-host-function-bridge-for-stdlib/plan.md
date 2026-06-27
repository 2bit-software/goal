# Implementation Plan — US-011 Host-function bridge for stdlib

## Files

### internal/interp/interp.go (modify)
- Add field `imports map[string]string` to `Interp` (local package name -> import path).
- In `New`, after constructing the struct, call `ip.registerImports()`.
- Add `registerImports()`: walk `ip.file.Imports`; for each spec compute local
  name (`spec.Name.Name` if present and not "_"/".", else last element of the
  unquoted `spec.Path.Value`) -> unquoted path.

### internal/interp/host.go (new)
- `type hostFunc func(args []Value) ([]Value, error)`
- `var hostFuncs = map[string]hostFunc{...}` keyed `"<path>.<Sym>"`:
  - `"fmt.Sprintf"` -> `StrVal(fmt.Sprintf(format, goArgs(rest)...))` (requires
    >=1 arg, first is a string).
  - `"fmt.Sprint"` -> `StrVal(fmt.Sprint(goArgs(all)...))`.
  - `"fmt.Println"` -> `fmt.Fprintln(os.Stdout, goArgs(all)...)`, returns no
    values. (US-023 will route through cap.)
  - `"fmt.Errorf"` -> `errVal(fmt.Errorf(format, goArgs(rest)...).Error())`.
  - `"errors.New"` -> `errVal(msg)` (requires exactly 1 string arg).
- `errVal(msg string) Value` = `StructVal("error", {"message": StrVal(msg)})`.
- `goArg(v Value) any`: KindInt/Float/String/Bool/Nil pass through to
  Go scalar/nil; an error struct (`TypeID == "error"`) becomes a real
  `errors.New(message)` so `%w`/`%v` render correctly; other composites render
  via `v.String()`.
- `goArgs([]Value) []any` maps `goArg` over a slice.
- `(*Interp) evalHostCall(sel *ast.SelectorExpr, call *ast.CallExpr, scope *Env)
  ([]Value, error)`:
  - path := ip.imports[X.Name]; key := path + "." + sel.Sel.Name.
  - Evaluate all args via ip.evalExpr.
  - fn, ok := hostFuncs[key]; if !ok return located named refusal:
    `interp: <pos>: unresolved imported call <key> (no host function registered)`
    using `sel.Pos().String()`.
  - else return fn(args).

### internal/interp/eval.go (modify)
- In `evalCallMulti`, before `tryMethodCall`, add: if `sel` is a SelectorExpr
  whose `X` is an `*ast.Ident` naming an imported package
  (`ip.imports[id.Name] != ""`) and that name is NOT shadowed by a scope binding
  (`scope.Lookup(id.Name)` errors), route to `ip.evalHostCall(sel, call, scope)`.

### internal/interp/host_test.go (new, package interp)
- `TestHostSprintf`: parse+resolve a goal program that returns
  `fmt.Sprintf("%s-%d", "x", 7)`, run it, assert "x-7".
- `TestHostErrorsNew` / `TestHostErrorf`: assert error struct message.
- `TestUnregisteredImportedCallNamedError`: a call to an imported-but-unshimmed
  symbol (e.g. `strings.ToUpper`) yields an error whose message names the
  missing `strings.ToUpper` and includes a location.
- `TestShadowedPackageNameFallsThrough` (optional): a local binding shadowing a
  package name does not route to the host bridge.

## Dependency ordering
1. interp.go imports registration (no dependents yet).
2. host.go registry + evalHostCall.
3. eval.go interception wiring.
4. tests.

## Testing strategy
- stdlib `testing` only (no testify). Drive real parsed+resolved programs through
  `New(...).Run()` or `ip.evalExpr(call, ip.root)` as the existing interp tests do.
- New stdlib imports introduced in interp: `os` (for Println). Confirm
  `go list -deps ./internal/interp` still excludes go/types, internal/backend,
  internal/typecheck (US-022 envelope).
