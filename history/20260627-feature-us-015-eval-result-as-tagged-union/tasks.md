# Implementation Tasks — US-015

## Task 1: Result tagged-union construction + match binding
**Status**: completed
**Files**: internal/interp/value.go, internal/interp/eval.go, internal/interp/interp.go
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2, FR-3
**Verify**: `go build ./...` && `go vet ./...`

### Instructions
- value.go: add `resultTypeID = "Result"`, `resultOkField = "value"`,
  `resultErrField = "error"` constants and a `payloadValue(v *Variant) (Value,
  bool)` helper returning the single field value of a one-field variant.
- eval.go `evalCallMulti`: after the builtin interception and before the
  host-package interception, add a block: if `call.Fun` is `*ast.SelectorExpr`
  whose `X` is `*ast.Ident` named "Result" and that name is not shadowed
  (`scope.Lookup` fails), call `ip.evalResultCtor(sel.Sel.Name, call, scope)`.
  `evalResultCtor` requires exactly one argument, evaluates it, and returns
  `VariantVal("Result", "Ok", {value: arg})` for `Ok`,
  `VariantVal("Result", "Err", {error: arg})` for `Err`, and a located,
  descriptive error for any other constructor name or wrong arity.
- interp.go `armScopeFor`: when `subj.Variant.TypeID == resultTypeID`, bind the
  unwrapped `payloadValue(subj.Variant)` (fallback to the whole variant if no
  single payload); otherwise keep binding the whole variant (enum behavior). This
  one seam serves both statement- and value-position match.

## Task 2: Tests over 03-result and 06-error-e shapes
**Status**: completed
**Files**: internal/interp/result_test.go
**Depends on**: Task 1
**Spec coverage**: all acceptance criteria
**Verify**: `go test ./internal/interp -count=1`

### Instructions
- New file, `package interp`, stdlib `testing` only (no testify). Reuse `newInterp`
  / `call` from call_test.go.
- Open-E (03-result): a `Result[Config, error]` program; assert Ok construction
  tag+payload, Err construction tag+payload (host error via errors.New), and a
  match whose Ok arm returns the unwrapped Config field and Err arm returns the
  unwrapped error message.
- Closed-E (06-error-e): a `Result[Config, ParseError]` program where Err carries
  the enum `ParseError.Empty`; assert the Err arm binds the enum variant and the
  Ok arm binds the Config — same runtime representation as open-E.
- Error cases: unknown `Result.<X>` ctor and wrong arity yield located errors.
- Final: run full `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
