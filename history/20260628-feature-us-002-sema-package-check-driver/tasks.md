# Implementation Tasks — US-002

## Task 1: Add the sema package driver
**Status**: completed
**Files**: `internal/sema/package.go`
**Depends on**: (none — composes existing sema seams + parser)
**Spec coverage**: FR-1, FR-2, FR-4
**Verify**: `go build ./...`

### Instructions
Create `internal/sema/package.go` (package `sema`). Import `goal/internal/ast`
and `goal/internal/parser`.

- `AnalyzePackageInDir(srcs []string, dir string) ([][]Diagnostic, error)`:
  delegate to `AnalyzePackageInDirWith(srcs, dir, nil)`, discard the `[]error`.
- `AnalyzePackageInDirWith(srcs []string, dir string, resolve DirResolver)
  ([][]Diagnostic, []error, error)`:
  1. Parse each `src` with `parser.ParseFile`; on error return `nil, nil, err`.
  2. `info := ResolvePackage(files)`.
  3. Aggregate `f.Imports` across all parsed files into one `[]*ast.ImportSpec`.
  4. `ferrs := EnrichForeign(info, imports, dir, resolve)`.
  5. Build `out := make([][]Diagnostic, len(files))`; `out[i] = Check(f, info)`
     in input order.
  6. Return `out, ferrs, nil`.

Mirror the doc-comment style of `internal/check/check.go` AnalyzePackageInDir.

## Task 2: Test the driver over a multi-file fixture
**Status**: completed
**Files**: `internal/sema/package_test.go`
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-2, FR-3
**Verify**: `go test ./internal/sema/ -count=1`

### Instructions
Create `internal/sema/package_test.go` (package `sema`, stdlib `testing`, no
testify). Model the fixtures on check/check_package_test.go and foreign_test.go.

- `TestAnalyzePackageInDirCrossFileExhaustiveness`: enum `Shape` in file A,
  non-exhaustive `match` over it in file B. Assert `len(out)==2`, file A has no
  `non-exhaustive-match` Error, file B does. (FR-1/FR-2.)
- `TestAnalyzePackageInDirForeignEnrichedDeriveFinding`: file A declares
  `type Target struct { ID string \n Extra string }`; file B has
  `import ext "example.com/ext"` + `derive func make(o *ext.Outer) Target`. Use a
  fake resolver mapping `example.com/ext` to the abs path of
  `../analyze/testdata/extpkg`. Assert file B has an `unsourced-field` Error
  (Target.Extra unsourced on ext.Outer). Control: run the same srcs through a
  resolver that errors (or nil enrichment) and assert the finding is instead the
  `unresolved-derive-type` Warning — proving the Error depends on enrichment.
- `TestAnalyzePackageInDirParseErrorReturned`: a malformed src returns non-nil err.

Helper `hasDiag(diags, severity, code) bool`.

## Coverage check
- FR-1: Task 1, Task 2. FR-2: Task 1, Task 2. FR-3: Task 2. FR-4: Task 1.
- Files: package.go (Task 1), package_test.go (Task 2). All covered.
