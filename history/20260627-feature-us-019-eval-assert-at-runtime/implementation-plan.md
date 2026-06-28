# Implementation Plan — US-019 Eval assert at runtime

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/assert.go` | `execAssert` (evaluate `assert`, no-op on true, located panic on false) + a self-contained `exprText` renderer for the condition message. |
| `internal/interp/assert_test.go` | Unit tests over a 10-assert-shaped program: true is a no-op, false panics with a located assertion message, message form formats, non-bool refused. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/interp/interp.go` | Add `case *ast.AssertStmt:` to `execStmt` routing to `ip.execAssert(s, scope)`. |

## Package Structure

```
internal/interp/
  interp.go        (execStmt switch gains the AssertStmt case)
  assert.go        (new — execAssert + exprText)
  assert_test.go   (new — tests)
```

No new package; no new dependency. internal/interp stays clear of
internal/backend / internal/typecheck / go/types (US-022 gate).

## Dependency Graph

1. `assert.go` `exprText` (pure AST -> string; no interp state).
2. `assert.go` `execAssert` (uses evalExpr, panicSignal, goArgs, exprText).
3. `interp.go` execStmt case (dispatches to execAssert).
4. `assert_test.go` (drives the whole thing).

## Interface Contracts

```go
// assert.go
func (ip *Interp) execAssert(s *ast.AssertStmt, scope *Env) error
func exprText(e ast.Expr) string   // readable source-ish text of a condition expr
```

`execAssert`:
- `cond, err := ip.evalExpr(s.Cond, scope)`; propagate err.
- `if cond.Kind != KindBool` -> `fmt.Errorf("interp: assert condition must be bool, got %s", cond.Kind)`.
- `if cond.Bool` -> return nil (no-op).
- else build the message:
  - `msg := "assertion failed: " + exprText(s.Cond)`
  - if `s.Msg != nil`: evaluate Msg + Args, `msg += ": " + fmt.Sprintf(format, goArgs(args)...)`.
  - prefix the source location: `s.Assert.String()`.
  - return `panicSignal{value: StrVal(<located message>)}`.

`exprText` recursively renders: Ident, BasicLit, ParenExpr, UnaryExpr,
BinaryExpr, SelectorExpr, CallExpr, IndexExpr; unknown nodes fall back to a
stable placeholder. Operators render via `token.Kind.String()`.

## Integration Points

- `internal/interp/interp.go` `execStmt`: new `case *ast.AssertStmt: return ip.execAssert(s, scope)`, placed beside the other statement cases (before `default`).
- `panicSignal` (interp.go) — reused verbatim; no boundary recovers it, so the panic surfaces to Run/host exactly like the `panic` builtin.
- `goArgs` (host.go) — reused for printf-message formatting, matching the backend.

## Testing Strategy

Mirror existing interp tests (stdlib `testing`, no testify). Build a
10-assert-shaped `package` source (bare assert + message-form assert), parse via
`parser.ParseFile`, resolve via `sema.Resolve`, construct `interp.New`, and run a
function via the existing `evalFn`/run helper. Cases:
- true condition: function returns normally (no panic error).
- false condition (bare): the returned error is a `panicSignal` whose message
  contains "assertion failed" and the condition text + location.
- false condition (message form): message contains the formatted text.
- non-bool condition: descriptive error (hand-built AssertStmt if needed).
