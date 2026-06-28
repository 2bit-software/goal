# Implementation Plan — US-004 Interpreter entry over AST and sema

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/interp.go` | The interpreter entry: `Interp` type, `New(*ast.File, *sema.Info)` constructor, and `Run() error` that locates and executes `func main`. |
| `internal/interp/interp_test.go` | Unit tests: trivial `package main\nfunc main() {}` parsed+resolved through parser+sema runs with no error; a program with no `main` yields a descriptive error. |

### Modified Files
None. (value.go / env.go are reused as-is.)

## Package Structure

```
internal/interp/
  value.go        (existing — runtime values)
  env.go          (existing — lexical scopes)
  interp.go       (NEW — entry/Run)
  *_test.go       (existing + interp_test.go NEW)
```

## Dependency Graph

1. internal/ast, internal/parser, internal/sema (existing front-end — unchanged)
2. internal/interp/value.go, env.go (existing — unchanged)
3. internal/interp/interp.go (NEW — depends on 1 and 2)
4. internal/interp/interp_test.go (NEW — depends on 3 + parser + sema)

## Interface Contracts

```go
// Interp runs a parsed, sema-resolved goal program.
type Interp struct {
    file *ast.File
    info *sema.Info
    root *Env
}

// New constructs an interpreter over the shared AST + sema front-end.
func New(file *ast.File, info *sema.Info) *Interp

// Run executes the program's `func main`. It returns a descriptive error if no
// top-level `func main` is declared; an empty body is a successful no-op.
func (ip *Interp) Run() error
```

## Integration Points

- Consumes `parser.ParseFile` output (`*ast.File`) and `sema.Resolve` output
  (`*sema.Info`) — the same artifacts `internal/backend` consumes. No dependency
  on internal/backend or any Go-lowered form (REWRITE-ARCHITECTURE.md §3.1).
- `Run` scans `file.Decls` for an `*ast.FuncDecl` with `Recv == nil` and
  `Name.Name == "main"`, opens a child Env off `root`, and walks the body
  statements (empty body → no-op).

## Testing Strategy

- `internal/interp/interp_test.go`, `package interp` (internal test, stdlib
  `testing` only, NO testify):
  - `TestRunTrivialMain`: parse+resolve `package main\nfunc main() {}`, call
    `New(file, info).Run()`, assert `err == nil`.
  - `TestRunMissingMainErrors`: parse+resolve a program with no `main`, assert
    `Run()` returns a non-nil error mentioning "main".
  - Optional: assert construction takes the AST+sema artifacts (compile-time via
    the call) — demonstrates the seam.
