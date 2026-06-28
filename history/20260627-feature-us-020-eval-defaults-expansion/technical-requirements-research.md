# Technical Requirements / Research — US-020

## Where it lives

- `internal/interp/eval.go`, `evalCompositeLit` — the struct (`*ast.Ident` type)
  case currently requires every element to be a `*ast.KeyValueExpr`. It must now
  also accept a `*ast.SpreadElement` whose `X` is the identifier `defaults`.

## AST shape

- `...defaults` parses to `*ast.SpreadElement{X: *ast.Ident{Name: "defaults"}}`
  inside the `*ast.CompositeLit.Elts` (see internal/ast/goal_expr.go and the
  backend's `compositeLit` handling in internal/backend/lower.go).
- A non-`defaults` spread (`...derive`) is US-021 — for now a descriptive refusal.

## Zero values

- Field declared types come from `ip.info.Structs[typeName]` (`[]sema.Field` with
  `Name` + `Type` string). The interpreter erases static types, so the runtime
  zero is computed from the sema type string, mirroring the backend's `zeroLit`
  (internal/backend/lower.go):
  - string -> "" ; bool -> false ; integer kinds -> 0 ; float kinds -> 0.0
  - pointer / map / chan / func / non-empty interface -> nil value
  - slice (`[]T`) -> empty slice (usable nil-slice equivalent)
  - named in-file struct -> recursively zero-filled struct value
- Only fields with a *safe* zero are ever defaulted; the sema field-completeness
  check (internal/sema CheckFields) already rejects a `...defaults` literal that
  omits an unsafe-zero field, so unsafe types do not reach this path in a valid
  program.

## Test

- Drive an 08-no-zero-value/defaults-shaped program through `newInterp` +
  `evalFn` (helpers in call_test.go / composite_test.go). Assert defaulted
  primitives are zero and explicitly set fields are preserved; assert a
  `...derive` (non-defaults) spread is refused.

## Dependency hygiene (US-022 gate)

- internal/interp must NOT gain a dependency on internal/backend, go/types, or
  internal/typecheck. Compute zeros locally from sema facts.
