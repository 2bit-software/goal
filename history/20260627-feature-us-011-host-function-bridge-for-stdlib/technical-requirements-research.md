# Technical Requirements / Research — US-011

## Context from the codebase

- `internal/interp` is the goscript tree-walking interpreter over the shared
  AST + sema front-end. It depends ONLY on `errors/fmt/strconv/sort` + goal's
  `internal/{ast,sema,token}`. The US-022 gate forbids `go/types`,
  `internal/backend`, `internal/typecheck` — so the host bridge must stay
  within the existing dependency envelope.
- A package-qualified call `fmt.Sprintf(...)` parses to an `*ast.CallExpr`
  whose `Fun` is an `*ast.SelectorExpr{X: *ast.Ident "fmt", Sel: "Sprintf"}`.
  Today `evalCallMulti` routes a selector call through `tryMethodCall` (struct
  receiver only) and then the generic function-value path, where evaluating
  `fmt` fails the scope lookup with a `*NotFoundError`.
- `file.Imports` (`[]*ast.ImportSpec`) gives the package set; each spec has an
  optional local `Name` and a `Path` string literal. Local name defaults to the
  last path element.
- `token.Pos{Offset, Line, Col}` already carries line/col and `String()`
  renders `"line:col"` — that supplies the "located" part of the refusal
  without importing `internal/check`.

## Design

- Add `imports map[string]string` (local name -> import path) to `Interp`,
  populated in `New` via `registerImports`.
- New file `internal/interp/host.go`:
  - A package-level `hostFuncs map[string]hostFunc` keyed by `"<path>.<Sym>"`
    (e.g. `"fmt.Sprintf"`). `hostFunc = func(args []Value) ([]Value, error)`.
  - `goArg(Value) any` converts a runtime Value to a Go `fmt` argument
    (primitives pass through; an error struct becomes a real `error`; composite
    kinds render via `Value.String()`).
  - Shims: `fmt.Sprintf`, `fmt.Sprint`, `fmt.Println` (writes os.Stdout —
    US-023 will route this through cap), `fmt.Errorf`, `errors.New`. Errors are
    represented as `StructVal("error", {"message": ...})`.
  - `evalHostCall(sel, call, scope)` resolves the package + symbol, evaluates
    args, dispatches, or returns a located, named refusal for an unregistered
    symbol.
- `evalCallMulti` gains an early interception: a selector call whose `X` is an
  imported, non-shadowed package name routes to `evalHostCall`.

## Testing

- stdlib `testing` only (no testify, project constraint).
- Parse + `sema.Resolve` a small goal program through `internal/parser` +
  `internal/sema`, run via the interpreter, assert `fmt.Sprintf` result and the
  named refusal for an unregistered symbol.
