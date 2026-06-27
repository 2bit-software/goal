# Technical Requirements / Research — US-019

## Existing seams

- `ast.AssertStmt{Assert, Cond, Comma, Msg, Args}` (internal/ast/goal_stmt.go) is
  the parsed node. Bare form: Msg nil, Args empty. Message form: Msg is a
  *BasicLit format string, Args the printf args; the message split only fires on
  the top-level comma after Cond.
- `internal/interp/interp.go execStmt` is the statement dispatch switch; add an
  `*ast.AssertStmt` case routing to a new `execAssert`.
- `panicSignal{value}` is the established loud non-local signal — NO call/loop/
  switch boundary recovers it; it propagates to Run / the host. `assert` reuses
  it (same stance as the `panic` builtin and the unreachable-match default).
- Booleans: `cond.Kind == KindBool` / `cond.Bool` (see execIf). A non-bool
  condition is a descriptive refusal.
- Message formatting reuses `goArgs` (host.go) + `fmt.Sprintf`, mirroring the
  backend's `assert` lowering (emit.go assertStmt).

## Constraints

- internal/interp must NOT depend on internal/backend / internal/typecheck /
  go/types (US-022 gate). The interp has no source bytes, so the condition text
  is rendered by a small self-contained expr printer in the interp (covers the
  forms that appear in assert conditions: Ident, BasicLit, Paren, Unary, Binary,
  Selector, Call, Index). The panic message is located via the assert position.
- The backend's panic message text is `assertion failed: <condText>[: <msg>]`;
  the interp mirrors the wording and adds the source location.
- Tests: stdlib `testing` only (no testify). Drive a 10-assert shaped program
  through interp.New + an evalFn-style helper, mirroring features/10-assert.
