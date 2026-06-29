# Tasks — US-002

## Task 1: Make sema and backend valid goal (rename `enum` identifiers)
**Depends on**: (none)
**Spec coverage**: FR-3 (green now)
**Verify**: `go build ./... && go test ./internal/sema/... ./internal/backend/...`

### Instructions
- `internal/sema/check.go`: rename the local variable `enum` to `enumDecl` in
  `checkOneMatch` (declaration `enum := info.Enums[enumName]` and all uses through
  the function) and in `missingVariants(enum *Enum, ...)` (param + body). `enum` is
  a goal reserved word; `enumDecl` is not. Pure rename, no behavior change.
- `internal/backend/emit.go`: rename the local `enum` to `enumIdent` in `variantLit`
  (`enum, ok := x.Enum.(*ast.Ident)` and its uses) and in the variant-name switch
  (`if enum, ok := b.Enum.(*ast.Ident); ...`). Leave `enumOf`, `x.Enum`, `.Enum`
  untouched (those are not the bare keyword).

## Task 2: Add the self-host smoke-gate harness
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-2, FR-4
**Verify**: `go build ./internal/selfhost/...`

### Instructions
- Create `internal/selfhost/selfhost.go`, `package selfhost`, implementing:
  - `var InScope = []string{"token","lexer","ast","parser","sema","project","pipeline","backend"}`
  - `func ReadPackage(dir string) (*project.Package, error)` — glob `dir/*.go`,
    skip `_test.go`, read each as a `project.File{Path, Name, Src}` (set `Name` to
    `<base>.goal` so `goName` yields clean `.go` names), derive `Name` from the
    first file's package clause via `parser.ParseFile`, set `Dir = dir`.
  - `func BuildTranspiled(layout map[string]*project.Package) error` — for each
    entry call `backend.TranspilePackage`, write `out.Files` into a temp module
    (`module goal`, `go 1.26`) under the module-relative key dir, then run
    `go build ./...`; return a package-identified error on transpile/build failure.
- Follow the temp-module pattern in `internal/corpus/package_runner.go`.

## Task 3: Add the gate tests
**Depends on**: Task 2
**Spec coverage**: FR-1, FR-2, FR-3
**Verify**: `go test ./internal/selfhost/...`

### Instructions
- Create `internal/selfhost/selfhost_test.go`, `package selfhost_test`:
  - `TestInScopePackagesTranspileAndBuild`: build `layout["internal/"+p] =
    ReadPackage("../"+p)` for each `p` in `InScope`; assert `BuildTranspiled` is nil.
  - `TestGateFailsOnNonCompilingTranspile`: build a layout with one in-memory
    package whose source transpiles but does not compile (e.g.
    `package broken\nfunc f() int { return }`); assert `BuildTranspiled` errors.
- stdlib `testing` only.
