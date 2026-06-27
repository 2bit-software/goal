# Implementation Report — US-004

## Task 1: Interpreter entry (Interp + New + Run) — DONE

### Files
- `internal/interp/interp.go` (new): `Interp{file,info,root}`, `New(*ast.File,
  *sema.Info)`, `Run() error`, `findMain`, `execBlock`, and `ErrNoMain`.
- `internal/interp/interp_test.go` (new): `TestRunTrivialMain`,
  `TestRunMissingMainErrors`, `TestConstructFromSharedFrontEnd`.

### What was built
- `New(file, info)` constructs the interpreter over the SHARED front-end
  artifacts (parsed `*ast.File` + resolved `*sema.Info`) and a root `Env`. No
  dependency on internal/backend or any Go-lowered form (FR-1, §3.1 seam).
- `Run()` locates the top-level `func main` (plain func, `Recv == nil`,
  `Name.Name == "main"`), opens a child scope, and walks the body. An empty body
  is a successful no-op (FR-2). Absent `main` returns the named sentinel
  `ErrNoMain` (FR-3).
- `execBlock` is the statement-dispatch seam US-005+ extends; today it no-ops.

### Verification
- `go build ./...` — pass
- `go vet ./...` — pass
- `go test ./internal/interp/ -count=1` — pass
- `go test ./... -count=1` — all packages pass

### Acceptance criteria
- Trivial `package main\nfunc main() {}` parsed+resolved through parser+sema runs
  with no error — covered by TestRunTrivialMain.
- Construction takes `*ast.File` + `*sema.Info`, no Go-lowered input — covered by
  TestConstructFromSharedFrontEnd.
- No `func main` yields a descriptive named error mentioning "main" — covered by
  TestRunMissingMainErrors.

### Deviations
None.
