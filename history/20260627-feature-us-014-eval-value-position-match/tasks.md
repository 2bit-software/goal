# Implementation Tasks

## Task 1: Add value-position match evaluation
**Status**: completed
**Files**: `internal/interp/interp.go`, `internal/interp/eval.go`, `internal/interp/value_match_test.go`
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2, FR-3, FR-4; all acceptance criteria
**Verify**: `go build ./...` && `go vet ./...` && `go test ./internal/interp/ -count=1` && `go test ./... -count=1`

### Instructions

Single cohesive task (small, tightly coupled change in one package).

1. In `internal/interp/interp.go`, factor two helpers out of the existing
   `execMatch`/`execArm`:
   - `func selectMatchArm(m *ast.MatchExpr, subj Value) (*ast.MatchArm, *ast.VariantPattern)`
     — walk `m.Arms`: return the `*ast.VariantPattern` arm whose
     `p.Variant.Name == subj.Variant.Tag`; remember a `*ast.RestPattern` arm and
     return it (with nil vp) if no variant arm matched; return `(nil, nil)` if
     neither exists.
   - `func armScopeFor(vp *ast.VariantPattern, subj Value, scope *Env) *Env`
     — open `scope.NewChild()`, and if `vp != nil && vp.Binding != nil` define
     `vp.Binding.Name` -> `subj`; return the scope.
   Rewrite `execMatch` to use `selectMatchArm` (raise the existing `panicSignal`
   "unreachable" message when it returns a nil arm) and `execArm` to use
   `armScopeFor`. Behaviour must be unchanged.

2. In `internal/interp/eval.go`:
   - Add `case *ast.MatchExpr: return ip.evalMatch(e, scope)` to `evalExpr`.
   - Add `func (ip *Interp) evalMatch(m *ast.MatchExpr, scope *Env) (Value, error)`:
     evaluate `m.Subject`; require `KindVariant` (else descriptive refusal);
     `arm, vp := selectMatchArm(m, subj)`; if `arm == nil` return
     `Value{}, panicSignal{...unreachable...}`; evaluate the arm body via
     `evalArmValue(arm.Body, armScopeFor(vp, subj, scope))`.
   - Add `func (ip *Interp) evalArmValue(body ast.Node, scope *Env) (Value, error)`:
     `case ast.Expr` -> `ip.evalExpr(b, scope)`; default -> descriptive refusal
     ("value-position match arm body must be an expression").

3. Add `internal/interp/value_match_test.go` (package `interp`, stdlib `testing`,
   no testify). Mirror `match_test.go`. Drive a 02-match-shaped program
   (`Shape{Point, Circle{radius}, Square{side}}`) through the existing
   `newInterp`/`evalFn` helpers. Cover:
   - `return match` function returns the right value per variant.
   - `x := match` and `var x = match` bind the right value per variant.
   - A payload arm computes from the binding (`Circle{c} => c.radius * c.radius`).
   - A `_` rest arm supplies the value when no variant arm matches.
   - A hand-built match over an uncovered tag -> `panicSignal` mentioning
     "unreachable" (errors.As at the Go boundary).

Run the verify commands; all must be green.
