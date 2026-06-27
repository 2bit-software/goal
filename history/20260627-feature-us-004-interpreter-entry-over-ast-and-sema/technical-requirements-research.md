# Technical Requirements & Research — US-004

## The seam

This is the load-bearing back-end seam from REWRITE-ARCHITECTURE.md §3.1: the
interpreter reads the typed AST + `*sema.Info`, NOT the Go backend's lowered
form. The constructor signature must therefore accept `*ast.File` + `*sema.Info`.

## Existing front-end APIs (to consume, not duplicate)

- `parser.ParseFile(src string) (*ast.File, error)` — internal/parser.
- `sema.Resolve(f *ast.File) *Info` — internal/sema (single-file); also
  `sema.ResolvePackage([]*ast.File)`.
- `ast.File.Decls []ast.Decl`; `ast.FuncDecl{Name *Ident, Recv *FieldList,
  Body *BlockStmt}`; a plain function has `Recv == nil`.
- Existing interp building blocks: `interp.Env` (env.go), `interp.Value`
  (value.go), `interp.NotFoundError`.

## Implementation shape

- New file `internal/interp/interp.go`:
  - `Interp` struct holding the `*ast.File`, `*sema.Info`, a root `*Env`.
  - `New(file *ast.File, info *sema.Info) *Interp` constructor.
  - `Run() error` — locates the top-level `func main` (plain func, no receiver),
    and executes its body. An absent `main` returns a descriptive error.
  - Statement evaluation is intentionally minimal for this story (an empty body
    is a no-op); richer eval lands in US-005+.
- Stdlib only, no testify (project constraint).

## Test plan

- `internal/interp/interp_test.go`: parse + resolve `package main\nfunc main() {}`
  via parser + sema, run through `New(...).Run()`, assert no error.
- Assert a program with no `main` returns a descriptive error.
