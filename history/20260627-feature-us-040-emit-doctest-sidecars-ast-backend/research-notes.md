# Research — US-040

## Canonical reference (splice path)
- internal/pass/doctests.go: ExtractDoctests(src) + RenderDoctests(src, tests)
  render a goal-shaped `_test.go`: `package <pkg>` + `import "testing"` + one
  `func TestDoctest_<fn>_<n>(t *testing.T)` per doctest with `got := <expr>`,
  `want := <expected>`, and a `got != want` t.Errorf guard.
- internal/pipeline/pipeline.go doctestFile: lowers that rendered goal source
  through the SAME passes + tables as the source, then gofmt-formats. Output.Test.

## AST-native equivalent
- US-023 already parsed doctests into ast.FuncDecl.Doc.Doctests
  ([]*ast.Doctest{Input string, Expected []string}). So extraction is a walk over
  file.Decls — no re-lexing.
- Render the same goal-shaped sidecar from those nodes, parse it back via
  parser.ParseFile, and emitFile(testFile, originalInfo). Passing the ORIGINAL
  info is what lets variant constructions / keyed literals in the doctest bodies
  lower (emit's variantLit/selectorExpr read info.Enums by name).

## Verification
- corpus.RunDoctest (sidecar vs golden, gofmt-normalized both sides) and
  corpus.RunDoctestExec (go test in temp module) already exist; drive the four
  features/11-doctests cases through corpus.TranspilerFunc(backend.Transpile).

Confidence: High — mirrors a known-good, build+vet+go-test-clean reference.
