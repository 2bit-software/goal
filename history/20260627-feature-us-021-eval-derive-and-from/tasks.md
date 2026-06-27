# Implementation Tasks — US-021

## Task 1: Register derive decls in the interpreter
**Status**: completed
**Files**: internal/interp/interp.go
**Depends on**: (none)
**Spec coverage**: FR-1 (foundation for derive evaluation)
**Verify**: `go build ./...`

### Instructions
- Add `derives map[string]*ast.FuncDecl` to the Interp struct.
- Initialize it in `New`.
- In `registerFuncs`, route a decl with `fn.Mod == ast.FuncDerive` into the
  derives map (keyed by fn.Name.Name) and `continue` — do NOT bind it as an
  ordinary callable. A bodyless derive (fn.Body == nil) must still be registered,
  so test the Mod before the existing `fn.Body == nil` skip.

## Task 2: Implement derive evaluation
**Status**: completed
**Files**: internal/interp/derive.go (new)
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-2, FR-3, FR-4, FR-5
**Verify**: `go build ./...` and `go vet ./...`

### Instructions
- `evalDerive(decl, call, scope)`: validate exactly one source arg, evaluate it,
  read srcName/srcType (first param), tgtType+fallible (deriveTargetType over
  results), overrides (deriveOverridesOf over body); call deriveConvert; return
  [target] for a total derive, [target, errVal-or-nil] for a fallible one (the
  error value is NilVal on success, the propagated conversion error on failure).
- `deriveConvert`: build the target field map. A nil pointer source returns the
  zero target. Evaluate overrides against a fresh root child binding srcName->src
  (skip `_`). For each un-overridden target field, find the same-named source
  field (findFieldFold), read its runtime value off the source struct, convert via
  convertFieldValue. Zero-fill any field left unset by a `_` skip. Unsourced field
  => descriptive located error naming derive + field.
- `convertFieldValue(v, sf, tf, fallible)`: identity (sf==tf) -> v; registry total
  -> callFunc(conv.Name, v); registry fallible (requires fallible) -> callFunc
  returns (val, err), propagate non-nil err; slice/array recursion (sliceElem/
  arrayElem + total element conv); map recursion (mapKeyVal + total element conv);
  nested in-file struct recursion (both sf and tf in Structs) -> recurse; pointer/
  Option -> loud refusal; otherwise loud "no conversion" refusal.
- Local helpers (no internal/backend import): deriveTargetType, deriveOverridesOf,
  deriveOverride type, findFieldFold, derefTypeName, sliceElem, arrayElem,
  mapKeyVal, and a typeName(ast.Expr) renderer matching sema key strings.
- Invoke registry conversions by name via root.Lookup + callFunc (mirror
  callConversion in eval.go).

## Task 3: Intercept derive calls in evalCallMulti
**Status**: completed
**Files**: internal/interp/eval.go
**Depends on**: Task 2
**Spec coverage**: FR-1
**Verify**: `go build ./...`

### Instructions
- In `evalCallMulti`, before the generic callee path (and after the
  Result/Option/host/method interceptions), add: if the callee is an `*ast.Ident`
  whose name is in `ip.derives` and is not shadowed by a scope binding, route to
  `ip.evalDerive(decl, call, scope)`.

## Task 4: Tests
**Status**: completed
**Files**: internal/interp/derive_test.go (new)
**Depends on**: Task 3
**Spec coverage**: all acceptance criteria
**Verify**: `go test ./internal/interp/ -run Derive -count=1` then full gates
`go build ./...`, `go vet ./...`, `go test ./... -count=1`

### Instructions
- Parse + sema-resolve a `derive_nested_struct`-shaped program; build New(file,
  info); evaluate `upgrade(Person{Name:"Ada", Home:Addr{Street:"Main", Zip:"90210"}})`
  via evalCallMulti against the root scope; assert PersonV2.Name=="Ada",
  Home.Street=="Main", Home.Zip is Code{v:"90210"} (registry bridge through nested
  struct recursion).
- Add a fallible-derive test: success returns Ok target + nil error; a conversion
  error path returns a non-nil error value.
- Add an unsourced-field refusal test (target field with no same-named source).
- stdlib testing only, no testify. Mirror defaults_test.go / question_test.go for
  the parse+resolve harness.
