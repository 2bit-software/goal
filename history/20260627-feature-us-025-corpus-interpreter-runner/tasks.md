# Implementation Tasks — US-025 Corpus Interpreter Runner

## Task 1: Interpreter doctest evaluator
**Status**: completed
**Files**: `internal/interp/doctest.go`, `internal/interp/doctest_test.go`
**Depends on**: (none)
**Spec coverage**: FR-2, FR-3, FR-5
**Verify**: `go test ./internal/interp -count=1`

### Instructions
- Create `internal/interp/doctest.go` (package interp):
  - `type DoctestFailure struct { Func, Input, Expected, Got string }`.
  - `func RunDoctests(src string) (failures []DoctestFailure, ran int, err error)`:
    - `parser.ParseFile(src)` (wrap parse error); `info := sema.Resolve(file)`;
      `ip := New(file, info)`.
    - pkg name from `file.Name.Name` (default "main").
    - For each `*ast.FuncDecl` with `Doc != nil`, each `*ast.Doctest`:
      `input := strings.TrimSpace(dt.Input)`,
      `want := strings.TrimSpace(strings.Join(dt.Expected, "\n"))`; skip if either empty.
      `ran++`. Parse the expr via the helper; `ip.evalExpr(expr, ip.root)` (eval error
      → return error naming func+input); if `val.String() != want` append a
      DoctestFailure{func name, input, want, got}.
  - `func parseDoctestExpr(pkg, input string) (ast.Expr, error)`: build
    `"package "+pkg+"\nfunc __doctest__() {\n\t__dt := "+input+"\n}\n"`, `parser.ParseFile`,
    lift `Decls[0].(*ast.FuncDecl).Body.List[0].(*ast.AssignStmt).Rhs[0]` with defensive
    shape checks (return a descriptive error on any mismatch).
- Create `internal/interp/doctest_test.go` (package interp): a doctest program with a
  correct `>>> add(2, 3)` / `5` (no failures, ran>0); a wrong-expected variant (one
  failure, fields populated); a doctest calling an undefined symbol (err != nil).
  stdlib testing only (no testify).

## Task 2: Corpus RunInterp runner + corpus tests
**Status**: completed
**Files**: `internal/corpus/interp_runner.go`, `internal/corpus/interp_runner_test.go`
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-3, FR-4
**Verify**: `go test ./internal/corpus -count=1`

### Instructions
- Create `internal/corpus/interp_runner.go` (package corpus), importing `goal/internal/interp`:
  - `func RunInterp(root string, c Case) error`:
    - Guard `c.Kind == KindDoctest` else descriptive wrong-kind error (mirror
      RunDoctestExec's wording).
    - Read `filepath.Join(root, filepath.FromSlash(c.Input))` (wrap read error w/ case ID).
    - `failures, ran, err := interp.RunDoctests(string(src))` (wrap err w/ case ID).
    - `ran == 0` → case-identified error ("produced no doctest examples").
    - `len(failures) > 0` → case-identified error listing each failure (func, input,
      expected, got).
    - else nil.
- Create `internal/corpus/interp_runner_test.go` (package corpus):
  - `TestInterpRunner`: load `manifestPath`, iterate `Kind==KindDoctest` cases, run each
    through `RunInterp(repoRoot, c)` in a sub-test, expect nil; `t.Fatalf` if zero ran.
  - `TestInterpRunnerWrongKind`: in-memory `Case{Kind: KindTranspile}` → error.
  - `TestInterpRunnerMutatedExpectedFails`: write a temp `.goal` with a wrong doctest
    expected, build an in-memory `Case{Kind: KindDoctest, Input: <relpath>}`, run with
    root=tempdir, assert an error mentioning the mismatch.
  - stdlib testing only (no testify).

## Task 3: Full verify gates
**Status**: completed
**Files**: (none — verification only)
**Depends on**: Task 2
**Spec coverage**: all ACs incl. native-only envelope
**Verify**: `go build ./...` && `go vet ./...` && `go test ./... -count=1`

### Instructions
- Run the prd.json verifyCommands in order; all green.
- Confirm `internal/interp` dep gate (TestInterpHasNoGoTypesOrTypecheckDep) still passes.
