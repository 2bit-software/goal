# Verify — Acceptance

Source of truth: business-spec.md acceptance criteria + PRD US-007.

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Function declarations are values | PASS | `New` registers each top-level plain func via `FuncDeclVal`; `TestFunctionDeclIsAValue` asserts `factorial` resolves to a callable func value. |
| Call binds args to params in fresh per-call scope; arg-count mismatch errors | PASS | `callFunc` flattens params, arity-checks, binds in `defScope.NewChild()`; `TestArgCountMismatch` asserts the named error. |
| Multiple return values; `q,r := f()` positional | PASS | `divmod` returns two; `TestMultiReturnAssignment` asserts q==3, r==2; single-value misuse refused by `TestMultiValueCallInSingleValueContext`. |
| Recursive factorial correct | PASS | `TestRecursiveFactorial` (0,1,5,6 -> 1,1,120,720). |
| Recursive fibonacci correct | PASS | `TestRecursiveFibonacci` (0,1,7,10 -> 0,1,13,55). |
| Loud refusals (undefined / non-func / arity) | PASS | `TestUndefinedCall` (*NotFoundError), `TestCallNonFunction` ("cannot call int"), `TestArgCountMismatch`. |

Project gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1` all green.
Dependency invariant (US-022): `go list -deps` confirms internal/interp pulls in
no go/types, internal/backend, or internal/typecheck despite the new `ast` import
in value.go.

## Assumptions
- A multi-value call is legal only as the sole RHS of a multi-assignment or the
  sole operand of `return`; elsewhere it must yield exactly one value (Go
  semantics). Stated in the spec, enforced in `evalCall`.
- `if`/`return` are implemented to the minimum the recursive ACs require; the full
  control-flow suite (for/switch/break/continue) remains US-008.
