# Technical Requirements / Research — US-042

## Engine default
- cmd/goal `parseFlags` defaults `engine = engineSplice`; flip to `engineAST`.
- Usage strings + flag summaries in main.go say "splice (default)"; update to
  reflect the AST default. Regenerate AI-KNOWLEDGE-BOOTSTRAP.md via
  `go run ./cmd/goal ai` (TestBootstrapGoldenMatches).
- main_test.go asserts default == engineSplice; flip to engineAST.

## Golden regeneration
- Exact tier = corpus.RunTranspile / RunDoctest (gofmt-normalized compare).
- Regenerate per file-mode KindTranspile case: write gofmt(backend Output.Go),
  or gofmt(Output.Test) when the golden is a doctest sidecar (isDoctestSidecar:
  imports "testing" + "func Test").
- Add a durable `-update-goldens` flag (mirrors parser's `-update-snapshots`) in
  package corpus to regenerate, then switch the exact-tier tests
  (TestTranspileRunner, TestDoctestRunner, TestCorpusTranspile, TestCorpusDoctest)
  from pipeline.Transpile to backend.Transpile.
- backend does not import corpus, so package-corpus tests may import backend
  (ast_gate_test.go already does). pipeline_test (external) may import backend too.

## Splice retention
- Splice stays exercised behaviorally (RunCompile/RunDoctestExec/RunPackage via
  pipeline.Transpile) and via the --engine=splice CLI flag — satisfying "available
  for one release".

## Manifest
- Golden CONTENT changes only; paths/counts unchanged, no manifest regen needed.

## Research findings (codebase-internal)
- US-041 already proved backend.Transpile succeeds on ALL file-mode transpile
  cases + doctest cases at the behavioral tier (ast_gate_test.go). So
  regenerating goldens from backend output cannot hit a transpile error.
- US-040 notes the 11-doctests sidecars are already byte-identical to goldens;
  those goldens likely won't change. Transpile goldens change (gensym `v` vs
  `__goal_v`, `type T int64` vs `= int64`, dropped `//` comments).
- No other golden comparison exists besides the 4 exact-tier tests + the
  AI-KNOWLEDGE-BOOTSTRAP.md guide golden.
- Confidence: High. The seam (Transpiler interface) makes the engine swap a
  one-line change per test; the -update flag mirrors the proven snapshot pattern.
