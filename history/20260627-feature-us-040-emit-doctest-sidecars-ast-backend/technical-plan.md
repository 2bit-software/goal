# Technical Plan — US-040 Emit doctest sidecars on new path

## Files
1. `internal/backend/doctest.go` (NEW)
   - `func emitDoctests(f *ast.File, info *sema.Info) (string, error)`:
     renders a goal-shaped `_test.go` from `f`'s structured doctests, parses it
     back via `parser.ParseFile`, then `emitFile(testFile, info)` — lowering the
     sidecar through the SAME emit path with the ORIGINAL file's resolved info.
     Returns "" when there are no doctests.
   - `func renderDoctests(f *ast.File) string`: walks `f.Decls` for
     `*ast.FuncDecl` with `Doc != nil && len(Doc.Doctests) > 0`, emits
     `package <f.Name>` + `import "testing"` + one `TestDoctest_<fn>_<n>` per
     doctest (`got := <Input>` / `want := <Expected>` / `got != want` t.Errorf).
     Mirrors internal/pass.RenderDoctests verbatim in shape. Returns "" if no
     doctest bodies were produced.

2. `internal/backend/backend.go` (EDIT)
   - `goBackend.Emit`: after `emitFile`, call `emitDoctests(file, info)` and set
     `Output.Test`. `Transpile` already gofmt-formats a non-empty `Output.Test`.

3. `internal/backend/backend_test.go` (EDIT, external package backend_test)
   - `TestASTEngineDoctestTier`: drives the 4 features/11-doctests cases through
     `corpus.RunDoctest` (sidecar vs golden) via `corpus.TranspilerFunc(backend.Transpile)`.
   - `TestASTEngineDoctestExecTier`: same cases through `corpus.RunDoctestExec`
     (behavioral go-test in a temp module); skipped under -short.
   - `TestASTEngineDoctestEnumLowering`: asserts the enum.goal sidecar lowers a
     variant construction (`Rejection(Rejection_MountNotGranted{Path: "/etc"})`).

## Ordering
doctest.go (new) -> backend.go Emit wiring -> tests. No circular deps:
backend already imports parser/ast/sema/pipeline; corpus does not import backend
(test is external package backend_test).

## Contract
- Input: parsed `*ast.File` + original `*sema.Info`.
- Output: formatted `_test.go` text in `pipeline.Output.Test`, byte-equal (modulo
  gofmt) to the splice sidecar for the plain cases; behaviorally identical for the
  enum case (gensym-free; doctest bodies use no generated temporaries).
