# Implementation Plan

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/value_match_test.go` | Unit tests for value-position match (`return match`, `x := match`, `var x = match`), payload-bearing arms, rest arm, and the loud unreachable default. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/interp/eval.go` | Add `case *ast.MatchExpr` to `evalExpr` -> new `evalMatch` (value-position dispatch returning the selected arm's Value) + `evalArmValue` helper. |
| `internal/interp/interp.go` | Factor `selectMatchArm` and `armScopeFor` helpers out of `execMatch`/`execArm` so statement- and value-position match share one dispatch + binding path. |

## Package Structure

```
internal/interp/
  eval.go               (modified — evalMatch, evalArmValue)
  interp.go             (modified — selectMatchArm, armScopeFor; execMatch/execArm reuse them)
  value_match_test.go   (new — value-position match tests)
```

No new packages; no dependency changes (the US-022 envelope is preserved — no
go/types, internal/backend, internal/typecheck imports introduced).

## Dependency Graph

1. `selectMatchArm` + `armScopeFor` helpers in interp.go (refactor; statement
   path keeps passing).
2. `evalMatch` + `evalArmValue` in eval.go (depends on the helpers from 1).
3. `evalExpr` `*ast.MatchExpr` case dispatch (depends on 2).
4. Tests (depend on 1-3).

## Interface Contracts

```go
// interp.go
// selectMatchArm picks the arm for a variant subject: the variant arm whose tag
// matches, else a rest (`_`) arm, else (nil, nil) -> caller raises unreachable.
func selectMatchArm(m *ast.MatchExpr, subj Value) (*ast.MatchArm, *ast.VariantPattern)

// armScopeFor opens a child scope and binds the arm's payload binding (if any)
// to the whole variant value.
func armScopeFor(vp *ast.VariantPattern, subj Value, scope *Env) *Env

// eval.go
// evalMatch evaluates a value-position match, returning the selected arm's value.
func (ip *Interp) evalMatch(m *ast.MatchExpr, scope *Env) (Value, error)

// evalArmValue evaluates a match arm body as an expression value.
func (ip *Interp) evalArmValue(body ast.Node, scope *Env) (Value, error)
```

## Integration Points

- `internal/interp/eval.go` `evalExpr` switch gains `case *ast.MatchExpr: return ip.evalMatch(e, scope)`.
- Value positions reach `evalExpr` already: `execReturn` (non-call result),
  `execAssign` (RHS), `execDecl` (spec values) in interp.go — no changes needed
  in those callers.
- `execMatch`/`execArm` are rewritten to call `selectMatchArm`/`armScopeFor`;
  their external behaviour (statement-position match) is unchanged.

## Testing Strategy

`internal/interp/value_match_test.go` (package `interp`, stdlib `testing`, no
testify). Drive a 02-match-shaped program through the existing `newInterp` /
`evalFn` helpers (call_test.go / composite_test.go), mirroring US-013's
`match_test.go`:

- `return match` function returns correct value per variant.
- `x := match` and `var x = match` bind correct value per variant.
- Payload arm computes from the bound payload (e.g. `Circle{radius} => radius*radius`).
- `_` rest arm supplies the value when no variant arm matches.
- A hand-built match over an uncovered tag raises `panicSignal` mentioning
  "unreachable".

Verify gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
