# Technical Plan — US-012 Eval enum construction

## Approach

Add enum-construction evaluation to `internal/interp/eval.go`, reusing the
existing tagged-union `Value` model. No value-model or sema change.

### 1. `evalExpr` dispatch

Add `case *ast.VariantLit: return ip.evalVariantLit(e, scope)`.

### 2. `evalVariantLit(vl, scope)`

- Resolve the enum name: `vl.Enum` must be an `*ast.Ident`; its name must be a
  key in `ip.info.Enums`. A qualified/absent enum ref is a descriptive refusal.
- Resolve the tag: `vl.Variant.Name` must be in the enum's `VSet`.
- Evaluate `vl.Args`:
  - `*ast.LabeledArg` → `fields[label] = eval(value)`.
  - positional `ast.Expr` → bind to the declared field at that index (from
    `enum.Variants[tag].Fields`), for robustness; an out-of-range positional is
    a refusal.
- Return `VariantVal(enumName, tag, fields)`.

### 3. `evalSelector` data-less interception

At the top of `evalSelector`, before evaluating the receiver: if `s.X` is an
`*ast.Ident` whose name is an enum in `ip.info.Enums`, the receiver name is NOT
bound in scope, and `s.Sel.Name` is in that enum's `VSet`, return
`VariantVal(enumName, s.Sel.Name, nil)`. Otherwise fall through to the existing
struct-field-read path.

### Helper

`ip.enumByName(name) (*sema.Enum, bool)` — nil-safe lookup of `ip.info.Enums`.

## Test plan

`internal/interp/enum_test.go` (new), parsing+resolving a 01-enums-shaped
program via `parser.ParseFile` + `sema.Resolve`, then:
- constructs `Status.Pending` (data-less) → asserts TypeID/Tag, empty fields.
- constructs `Status.Active(since: now())` → asserts Tag + reads `since` field.
- constructs `Status.Cancelled(reason: "timeout", at: now())` → asserts Tag +
  reads both fields.
- asserts an unknown variant and an unknown enum are descriptive refusals.

Run: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.

## Risk / out of scope

- Positional `Enum.Variant(x)` (all-positional, no label) parses to a CallExpr,
  not a VariantLit — Result/Option construction is US-015/US-016. Out of scope.
- Match destructuring (VariantPattern) is US-013. Out of scope.
