# Implementation Plan — US-015 Eval Result as tagged union

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/result_test.go` | Unit tests over 03-result (open-E) and 06-error-e (closed-E) shapes: Ok/Err construction tags + payloads, and match-arm payload/error binding in both statement and value position. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/interp/value.go` | Add canonical Result payload field-name constants and a `payloadValue(*Variant)` helper returning the single unwrapped payload of a sum-payload variant. |
| `internal/interp/eval.go` | In `evalCallMulti`, intercept `Result.Ok(x)` / `Result.Err(e)` (selector call, receiver `Result` not shadowed) and construct the tagged-union value via `VariantVal`. Exactly one arg each; unknown ctor / wrong arity are located refusals. |
| `internal/interp/interp.go` | In `armScopeFor`, unwrap the single payload for Result variants (`TypeID == "Result"`) so an `Ok`/`Err` arm binds the inner value/error, not the whole variant; enum bindings keep binding the whole variant. |

## Package Structure

All changes are within the existing `internal/interp` package — no new packages.

```
internal/interp/
  value.go        (+ resultOk/resultErr field consts, payloadValue helper)
  eval.go         (+ Result.Ok/Err construction interception in evalCallMulti)
  interp.go       (+ Result payload unwrap in armScopeFor)
  result_test.go  (new tests)
```

## Dependency Graph

1. `value.go` — field-name constants + `payloadValue` helper (no deps).
2. `eval.go` — `Result.Ok/Err` construction uses `VariantVal` + the field consts.
3. `interp.go` — `armScopeFor` unwrap uses `payloadValue` + the `TypeID == "Result"` check.
4. `result_test.go` — exercises 1–3 end to end.

## Interface Contracts

```go
// value.go
const (
    resultTypeID   = "Result"
    resultOkField  = "value" // payload of Result.Ok
    resultErrField = "error" // payload of Result.Err
)

// payloadValue returns the single payload value of a sum-payload variant
// (Result/Option), ok=false when the variant carries other than one field.
func payloadValue(v *Variant) (Value, bool)

// eval.go — inside evalCallMulti, before the host/method interceptions:
//   if call.Fun is *ast.SelectorExpr{X: *ast.Ident "Result"} and "Result"
//   is not shadowed in scope -> ip.evalResultCtor(sel.Sel.Name, call, scope)
func (ip *Interp) evalResultCtor(ctor string, call *ast.CallExpr, scope *Env) ([]Value, error)

// interp.go — armScopeFor binds the unwrapped payload for a Result variant:
//   if subj.Variant.TypeID == resultTypeID, bind payloadValue(subj.Variant)
//   (fallback: whole variant); else bind the whole variant (enum behavior).
```

## Integration Points

- `evalCallMulti` (internal/interp/eval.go): new interception block placed after
  the builtin check and before the host-package / method interceptions, mirroring
  the existing "not shadowed in scope" guard pattern.
- `armScopeFor` (internal/interp/interp.go): the single shared bind seam used by
  both `execMatch` (statement position) and `evalMatch` (value position), so the
  unwrap applies to both for free.
- `selectMatchArm` is UNCHANGED — tag dispatch (`Ok`/`Err`) already works over a
  Result subject.

## Testing Strategy

`internal/interp/result_test.go`, package `interp`, stdlib `testing` only (no
testify), reusing `newInterp` / `call` helpers from call_test.go:

- `TestResultOkConstruction` / `TestResultErrConstruction`: a function returning
  `Result.Ok(...)` / `Result.Err(...)` evaluates to a KindVariant with TypeID
  "Result" and tag "Ok"/"Err", and the payload reads back.
- `TestResultMatchBindsOkPayloadOpenE` / `TestResultMatchBindsErrOpenE`: a
  03-result-shaped program (`Result[Config, error]`, Err carries a host error via
  errors.New) where a match Ok arm returns the unwrapped Config field and the Err
  arm returns the unwrapped error message.
- `TestResultMatchClosedE`: a 06-error-e-shaped program (`Result[Config,
  ParseError]`, Err carries the enum `ParseError.Empty`) asserting the Err arm
  binds the enum variant and the Ok arm binds the Config.
- `TestResultUnknownCtorIsRefused` / `TestResultCtorArityIsRefused`: located
  descriptive errors.

Verify gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
