# Technical Requirements — US-040

- The parser (US-023) already attaches structured doctests to
  `ast.FuncDecl.Doc.Doctests` (each `Doctest{Input, Expected []string}`). The AST
  backend extracts from those nodes — AST-native, no re-lexing of source text.
- Mirror the splice path (internal/pipeline.doctestFile + internal/pass.RenderDoctests):
  render a goal-shaped `_test.go` (package clause + `import "testing"` + one
  `TestDoctest_<fn>_<n>` per doctest with `got := <Input>` / `want := <Expected>` /
  inequality check), then lower it through the SAME emit path.
- Lower the rendered sidecar by parsing it back to an *ast.File and calling
  emitFile with the ORIGINAL file's *sema.Info, so variant constructions / keyed
  literals in doctest bodies lower against the source's resolved enums/structs.
- Wire into goBackend.Emit so backend.Transpile populates Output.Test; Transpile
  already gofmt-formats Output.Test when non-empty.
- Verify with corpus.RunDoctest (sidecar text vs golden) and corpus.RunDoctestExec
  (behavioral: go test in a temp module) over the 4 11-doctests cases, driven
  through corpus.TranspilerFunc(backend.Transpile) in the external backend_test pkg.
