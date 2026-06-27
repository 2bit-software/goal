# Implementation Plan — US-025 Corpus Interpreter Runner

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/doctest.go` | Exported `RunDoctests(src string) ([]DoctestFailure, error)`: parse + resolve + build interp, evaluate each `>>>` doctest expression, return per-example mismatches (loud parse/eval errors). Plus the `DoctestFailure` type. |
| `internal/corpus/interp_runner.go` | Exported `RunInterp(root string, c Case) error`: load a doctest (Mode=file) case, run it through `interp.RunDoctests`, turn any failure/zero-doctest/wrong-kind into a case-identified error. |
| `internal/interp/doctest_test.go` | Unit tests for `RunDoctests` (pass on a doctest program, fail loudly on a wrong expected value, surface an eval error). |
| `internal/corpus/interp_runner_test.go` | Runs every doctest corpus case through `RunInterp` (expect pass), asserts a mutated-expected case fails, asserts a wrong-kind case is refused. |

### Modified Files
None required (no production code changes to existing files; the new runner composes existing seams).

## Implementation Detail

### `internal/interp/doctest.go`
- `type DoctestFailure struct { Func, Input, Expected, Got string }` with an
  `Error()`-style `String()` for messages.
- `RunDoctests(src string) ([]DoctestFailure, error)`:
  1. `file, err := parser.ParseFile(src)` — wrap parse error.
  2. `info := sema.Resolve(file)`; `ip := New(file, info)`.
  3. Determine package name (`file.Name.Name`, default "main") for the expr wrapper.
  4. For each `*ast.FuncDecl` with `Doc != nil`, for each `*ast.Doctest`:
     - `input := strings.TrimSpace(dt.Input)`, `want := strings.TrimSpace(strings.Join(dt.Expected, "\n"))`; skip if either empty.
     - Parse the input expression: build `package <pkg>\nfunc __doctest__() {\n\t__dt := <input>\n}\n`, `parser.ParseFile` it, and lift the body's first stmt `*ast.AssignStmt`'s `Rhs[0]`. A structural failure here is a returned error naming the function + input.
     - `val, err := ip.evalExpr(expr, ip.root)` — eval error → returned error naming the function + input.
     - `got := val.String()`; if `got != want` append a `DoctestFailure`.
  5. Return `(failures, nil)`.
- Helper `parseDoctestExpr(pkg, input string) (ast.Expr, error)` isolates the wrap+lift.

### `internal/corpus/interp_runner.go`
- `RunInterp(root string, c Case) error`:
  1. Guard `c.Kind == KindDoctest` else descriptive wrong-kind error.
  2. (Doctest cases are Mode=file.) Read `filepath.Join(root, filepath.FromSlash(c.Input))`; wrap read error with case ID.
  3. `failures, err := interp.RunDoctests(string(src))`; wrap err with case ID.
  4. If `len(failures) == 0 && noDoctests`, fail loudly: a doctest case that yielded
     zero examples is an error (detect via a count returned alongside, or re-derive).
     Implementation: have `RunDoctests` also return the evaluated count, or expose
     it; simplest is `RunInterp` treats "ran == 0" as an error. To keep the API
     small, `RunDoctests` returns `(failures []DoctestFailure, ran int, err error)`.
  5. If `len(failures) > 0`, return a case-identified error listing each failure
     (func, input, expected, got).
  6. nil on all-pass.

### Tests
- `internal/interp/doctest_test.go` (package interp): a small doctest program string
  (`/// >>> add(2, 3)` etc.), assert `RunDoctests` returns no failures; a mutated
  program with a wrong expected returns one failure with the right fields; a program
  whose doctest calls an undefined symbol returns an error.
- `internal/corpus/interp_runner_test.go` (package corpus): load `manifestPath`,
  iterate `Kind==KindDoctest` cases, run each through `RunInterp(repoRoot, c)`,
  expect nil; fail loudly if zero doctest cases. Plus a unit test that constructs an
  in-memory wrong-kind Case and asserts an error, and (using a temp file) a mutated
  doctest fails. Keep it in `package corpus` (no corpus-runner import cycle: RunInterp
  imports interp, not pipeline/check, so an internal test is fine).

## Verification

- `go build ./...`
- `go vet ./...`
- `go test ./... -count=1`
- `go list -deps goal/internal/corpus | grep -E 'go/types|internal/typecheck'` — the
  interp edge must not drag go/types into corpus via interp (corpus already depends on
  typecheck transitively through check/pipeline, so the meaningful gate is the existing
  `internal/interp` TestInterpHasNoGoTypesOrTypecheckDep, which still holds).

## Risks / Notes

- The interp has no `ParseExpr`; the wrap-and-lift around `:=` is the established way
  to obtain an expression node from `parser.ParseFile` (the only exported parse entry).
- `Value.String()` already renders int/string/bool in the doctest golden's Go-literal
  spelling, so it is the correct behavioral oracle for the four committed doctest cases.
