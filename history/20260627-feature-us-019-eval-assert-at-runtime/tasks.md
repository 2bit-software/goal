# Implementation Tasks

## Task 1: Implement execAssert + exprText and wire into execStmt
**Status**: pending
**Files**: `internal/interp/assert.go` (new), `internal/interp/interp.go` (modify)
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2, FR-3, FR-4
**Verify**: `go build ./... && go vet ./...`

### Instructions
- Create `internal/interp/assert.go`:
  - `func (ip *Interp) execAssert(s *ast.AssertStmt, scope *Env) error`:
    - Evaluate `s.Cond` via `ip.evalExpr`; propagate any error.
    - If `cond.Kind != KindBool`: return
      `fmt.Errorf("interp: assert condition must be bool, got %s", cond.Kind)` (FR-4).
    - If `cond.Bool`: return nil (no-op, FR-1).
    - Else build `body := "assertion failed: " + exprText(s.Cond)`; if
      `s.Msg != nil`, evaluate Msg + each Arg, append
      `": " + fmt.Sprintf(format, goArgs(args)...)` (FR-3). Prefix the source
      location `s.Assert.String()`. Return
      `panicSignal{value: StrVal(located)}` (FR-2) — reusing the existing
      loud signal so no boundary recovers it.
  - `func exprText(e ast.Expr) string`: a small recursive renderer covering
    Ident, BasicLit (use .Value), ParenExpr, UnaryExpr, BinaryExpr,
    SelectorExpr, CallExpr, IndexExpr; operators via `token.Kind.String()`;
    unknown/nil -> a stable placeholder ("<expr>"). Self-contained — imports only
    `goal/internal/ast` (+ token for Kind.String, already available transitively
    via node fields) and `strings`/`fmt`. MUST NOT import internal/backend.
- Modify `internal/interp/interp.go` `execStmt`: add
  `case *ast.AssertStmt: return ip.execAssert(s, scope)` before `default`.

## Task 2: Tests over a 10-assert shape
**Status**: pending
**Files**: `internal/interp/assert_test.go` (new)
**Depends on**: Task 1
**Spec coverage**: AC test (true no-op + false panic), FR-1..FR-4
**Verify**: `go test ./internal/interp/ -run Assert -count=1`

### Instructions
- stdlib `testing` only, no testify. Reuse the existing `evalFn`/run helper used
  by the other interp tests (do not redeclare it).
- Drive a 10-assert-shaped program modeled on
  features/10-assert/examples/{bank,message}.goal.
- Cases:
  - true condition -> no error / normal completion.
  - false bare condition -> error is a `panicSignal` whose message contains
    "assertion failed" + the condition text + a location.
  - false message-form condition -> message contains the formatted text.
  - non-bool condition -> descriptive "must be bool" error (hand-build the
    AssertStmt if the parser would otherwise reject a non-bool literal).
