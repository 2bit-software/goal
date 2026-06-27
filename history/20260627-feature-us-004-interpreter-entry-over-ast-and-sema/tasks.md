# Implementation Tasks — US-004

## Task 1: Interpreter entry (Interp + New + Run) with tests
**Status**: pending
**Files**: `internal/interp/interp.go` (new), `internal/interp/interp_test.go` (new)
**Depends on**: (none — builds on existing value.go/env.go + front-end)
**Spec coverage**: FR-1, FR-2, FR-3; all acceptance criteria
**Verify**: `go build ./...` && `go vet ./...` && `go test ./internal/interp/ -count=1`

### Instructions
- Create `internal/interp/interp.go`, `package interp`:
  - Import `internal/ast` and `internal/sema` (use the module path prefix the
    other internal packages use — check an existing file like value.go's
    neighbors / sema imports).
  - Define `type Interp struct { file *ast.File; info *sema.Info; root *Env }`.
  - `func New(file *ast.File, info *sema.Info) *Interp` — store args, set
    `root: NewEnv()`.
  - `func (ip *Interp) Run() error`:
    - Scan `ip.file.Decls` for `*ast.FuncDecl` with `Recv == nil` and
      `fn.Name != nil && fn.Name.Name == "main"`.
    - If none found, return a descriptive error naming "main" (e.g.
      `errors.New("interp: no func main")` or a typed error).
    - Otherwise open `ip.root.NewChild()` and walk `fn.Body.List` (empty body →
      no-op; full statement evaluation is US-005+). Return nil on success.
  - Keep it minimal and dependency-free beyond ast/sema/stdlib (NO internal/backend).
- Create `internal/interp/interp_test.go`, `package interp`, stdlib `testing`
  only (NO testify):
  - `TestRunTrivialMain`: `src := "package main\nfunc main() {}\n"`;
    `f, err := parser.ParseFile(src)`; fatal on err; `info := sema.Resolve(f)`;
    `if err := New(f, info).Run(); err != nil { t.Fatalf(...) }`.
  - `TestRunMissingMainErrors`: parse+resolve `package main\nfunc other() {}\n`;
    assert `Run()` returns a non-nil error whose message contains "main".
  - Import `internal/parser` and `internal/sema` with the module path prefix.
