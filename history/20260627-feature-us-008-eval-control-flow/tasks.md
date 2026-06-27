# Implementation Tasks — US-008 Eval control flow

## Task 1: Control-flow evaluation in the interpreter
**Status**: completed
**Files**: internal/interp/interp.go
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2, FR-3, FR-4
**Verify**: `go build ./... && go vet ./...`

### Instructions
In internal/interp/interp.go:
- Add two error-sentinel types mirroring `returnSignal`:
  - `breakSignal struct{}` with `Error()` returning
    `"interp: break outside loop or switch"`.
  - `continueSignal struct{}` with `Error()` returning
    `"interp: continue outside loop"`.
- Extend `execStmt`'s type switch with cases:
  - `*ast.ForStmt` -> `execFor`
  - `*ast.SwitchStmt` -> `execSwitch`
  - `*ast.BranchStmt` -> `execBranch`
  - `*ast.IncDecStmt` -> `execIncDec`
  - `*ast.BlockStmt` -> `ip.execBlock(s, scope.NewChild())`
- `execFor`: open `loopScope := scope.NewChild()`; run `s.Init` if non-nil in
  loopScope; loop: evaluate `s.Cond` in loopScope (nil => true; non-bool =>
  descriptive error); run body in `loopScope.NewChild()`; use `errors.As` to
  recover `continueSignal` (fall through to Post) and `breakSignal` (return nil);
  let returnSignal and real errors propagate; run `s.Post` in loopScope after each
  iteration (including after a continue).
- `execSwitch`: open `swScope := scope.NewChild()`; run `s.Init`; evaluate
  `s.Tag` if present. Walk `s.Body.List` CaseClauses: remember the default
  (List==nil); for each non-default clause, a tagged switch matches when a list
  expr `.Equal(tag)`, a tagless switch matches when a list expr is bool true.
  Run the first matched clause body (else the default) in `swScope.NewChild()`;
  recover `breakSignal` (stop, return nil). No fallthrough. continueSignal and
  returnSignal propagate out.
- `execBranch`: `token.BREAK` -> `return breakSignal{}`; `token.CONTINUE` ->
  `return continueSignal{}`; otherwise a descriptive "unsupported branch" error.
- `execIncDec`: target must be `*ast.Ident` (else descriptive error); Lookup the
  current value; apply `applyBinary(token.ADD/SUB, cur, one)` where `one` is
  `IntVal(1)` for KindInt or `FloatVal(1)` for KindFloat; Assign the result back.
- Follow the existing comment/style discipline of execIf/execReturn.

## Task 2: Control-flow tests
**Status**: completed
**Files**: internal/interp/control_test.go
**Depends on**: Task 1
**Spec coverage**: all acceptance criteria
**Verify**: `go test ./internal/interp/ -count=1`

### Instructions
New file `internal/interp/control_test.go`, `package interp`, stdlib `testing`
only (no testify). Mirror eval_test.go/call_test.go helpers (parser.ParseFile +
sema.Resolve + New, or build AST and evaluate the main body against a child
scope, reading vars via Lookup). Cover:
- summation three-clause loop total;
- condition-only loop and `for {}`+break;
- continue skips remainder of body;
- tagged switch dispatch + default fallback;
- tagless switch first-true case;
- break inside switch exits only the switch (enclosing loop continues);
- nested block scoping (inner var absent from outer scope -> Lookup error);
- if/else chain regression;
- error cases: non-bool for cond; break outside any loop.
