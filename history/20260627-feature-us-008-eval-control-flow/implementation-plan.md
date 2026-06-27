# Implementation Plan — US-008 Eval control flow

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/control_test.go` | In-package (`package interp`) table-driven tests for for/switch/block/break/continue + if/else regression. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/interp/interp.go` | Add `breakSignal`/`continueSignal` error-sentinel types; extend `execStmt` to dispatch ForStmt, SwitchStmt, BlockStmt, BranchStmt, IncDecStmt; add `execFor`, `execSwitch`, `execBranch`, `execIncDec`. |

No changes to value.go/env.go/eval.go are required: Env already has
NewChild/Define/Lookup/Assign; applyBinary/Value.Equal already exist for switch
case comparison and `i++`.

## Package Structure

```
internal/interp/
  interp.go        (modified: control-flow statement evaluation)
  control_test.go  (new: control-flow tests)
```

## Dependency Graph

1. `breakSignal` / `continueSignal` sentinel types (no deps).
2. `execFor` / `execSwitch` / `execBranch` / `execIncDec` (depend on 1 + existing
   execBlock/execStmt/evalExpr/applyBinary).
3. `execStmt` dispatch cases wiring 2 in.
4. `control_test.go` (depends on 1-3).

## Interface Contracts

```go
// Non-local loop/switch control signals (mirroring returnSignal).
type breakSignal struct{}
func (breakSignal) Error() string { return "interp: break outside loop or switch" }

type continueSignal struct{}
func (continueSignal) Error() string { return "interp: continue outside loop" }

func (ip *Interp) execFor(s *ast.ForStmt, scope *Env) error
func (ip *Interp) execSwitch(s *ast.SwitchStmt, scope *Env) error
func (ip *Interp) execBranch(s *ast.BranchStmt, scope *Env) error
func (ip *Interp) execIncDec(s *ast.IncDecStmt, scope *Env) error
```

Semantics:
- execFor: open `loopScope := scope.NewChild()`; run Init in loopScope; loop while
  Cond (nil => true; non-bool => error) evaluated in loopScope; each body runs in
  `loopScope.NewChild()`. Recover `continueSignal` (run Post, continue) and
  `breakSignal` (stop). Post runs in loopScope after a normal or continue
  iteration. returnSignal and real errors propagate.
- execSwitch: open `swScope := scope.NewChild()`; run Init; evaluate Tag if
  present. Iterate CaseClauses: a clause with List==nil is the default (remembered,
  run last if nothing matched). For a tagged switch, a case matches when any
  list expr `.Equal` the tag; for a tagless switch, when any list expr is bool
  true. Run the matched clause's Body in `swScope.NewChild()`; recover
  `breakSignal` (stop). No fallthrough. continueSignal/returnSignal propagate.
- execBranch: token.BREAK => return breakSignal{}; token.CONTINUE =>
  continueSignal{}; anything else (goto/fallthrough) => descriptive error.
- execIncDec: read ident via Lookup, apply +1/-1 via applyBinary(ADD/SUB, cur,
  IntVal(1)/FloatVal for float), Assign back. Non-ident target => error.

## Integration Points

`execStmt` in interp.go gains cases: `*ast.ForStmt -> execFor`,
`*ast.SwitchStmt -> execSwitch`, `*ast.BranchStmt -> execBranch`,
`*ast.IncDecStmt -> execIncDec`, `*ast.BlockStmt -> execBlock(s, scope.NewChild())`.

## Testing Strategy

`control_test.go`, `package interp`, stdlib `testing` only (no testify). Helpers:
parse a small program with `parser.ParseFile`, resolve with `sema.Resolve`,
`New(...)`, then either `Run()` or evaluate the main body against a child scope and
read a variable via `Lookup`. Cases:
- Summation three-clause loop -> assert sum.
- Condition-only loop and `for {}`+break -> assert termination + accumulator.
- continue skips body remainder -> assert partial accumulation.
- Tagged switch dispatch + default fallback -> assert selected arm effect.
- Tagless switch first-true case -> assert selected arm.
- break inside switch exits only the switch (enclosing loop continues).
- Nested block scoping: var inside block absent from outer scope (Lookup error).
- if/else chain regression.
- Error cases: non-bool for cond; break outside loop.
