# Research — US-004

This story is entirely internal to the goal codebase; no external/library
research applies. Findings come from reading the existing front-end and interp
packages.

## Confirmed front-end entry points

- `parser.ParseFile(src string) (*ast.File, error)` — internal/parser/parser.go:53.
- `sema.Resolve(f *ast.File) *Info` — internal/sema/resolve.go:53 (populates
  Enums/Sealed/Structs/Methods/etc.). `sema.New()` returns an EMPTY Info; the
  test should use `Resolve` to exercise the real front-end as the AC demands
  ("parses and resolves ... through internal/parser + internal/sema").
- `ast.File.Decls []ast.Decl`; the entry point is an `*ast.FuncDecl` with
  `Name.Name == "main"` and `Recv == nil` (plain function, not a method).
- `ast.BlockStmt.List []ast.Stmt` is the body; an empty body is a valid no-op.

## Existing interp building blocks to reuse

- `interp.NewEnv()` / `*Env` (env.go) — the root scope for the run.
- `interp.Value` (value.go) — runtime values; not exercised by an empty main but
  the Run path is shaped to return values later.

## Decision

Add `internal/interp/interp.go` with `Interp` + `New(*ast.File, *sema.Info)` +
`Run() error`. Keep statement evaluation minimal (empty body = no-op); US-005+
fill in expression/statement eval. Absent `main` => descriptive error, never a
silent success.

**Confidence**: High — all APIs verified by reading source.
**Open questions**: none for this seam-only story.
