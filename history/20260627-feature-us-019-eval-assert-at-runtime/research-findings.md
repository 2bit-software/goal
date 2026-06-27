# Research Findings — US-019

This is an internal-codebase task (extend the existing tree-walking interpreter);
no external/web research is needed. Findings from reading the codebase:

## Summary

The `assert` statement is already fully represented in the AST
(`ast.AssertStmt`) and lowered by the Go backend
(`internal/backend/emit.go assertStmt`). The interpreter simply has no case for
it yet — `execStmt` returns `interp: unsupported statement *ast.AssertStmt`.

The implementation mirrors three established interp patterns:
1. **Bool-condition refusal**, exactly like `execIf` (`cond.Kind != KindBool` ->
   descriptive error).
2. **Loud non-local panic**, exactly like the `panic` builtin and the
   unreachable-match default — `panicSignal{value}` rides the `(… error)` channel
   and is recovered by no boundary, surfacing to the host.
3. **printf-message formatting**, exactly like host.go's fmt shims — `goArgs` +
   `fmt.Sprintf`.

The backend panic message is `assertion failed: <condText>[: <fmt msg>]`. The
interp has no source bytes (it works off the AST per the US-022 native-only
gate), so a small self-contained expr-to-text renderer reproduces the condition
text and the panic is additionally located via the assert source position.

## Confidence

High — the node, the backend reference lowering, the fixtures
(features/10-assert/examples/{bank,message,multiple}.goal), and the three reused
interp seams are all present and read.

## Open questions

None. The only design choice is the condition-text source: rendered from the AST
(chosen) rather than sliced from source bytes the interp does not hold.
