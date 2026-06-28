# Implementation Plan — US-042

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| internal/corpus/update_goldens_test.go | `-update-goldens` flag + TestUpdateGoldens: regenerates every file-mode transpile/doctest golden from backend.Transpile (gofmt-normalized; Output.Test for doctest sidecars, else Output.Go). Mirrors parser's `-update-snapshots`. |

### Modified Files
| File | Change |
|------|--------|
| cmd/goal/main.go | parseFlags default `engine = engineAST`; usage Usage/Summary strings: AST is default, splice is legacy opt-in. |
| cmd/goal/main_test.go | Default-engine assertion flips to engineAST. |
| internal/corpus/runner_test.go | TestTranspileRunner transpiler -> backend.Transpile. |
| internal/corpus/doctest_runner_test.go | TestDoctestRunner transpiler -> backend.Transpile. |
| internal/pipeline/pipeline_test.go | TestCorpusTranspile + TestCorpusDoctest transpiler -> backend.Transpile. |
| features/**/*.go.expected, testdata/**/*.go.expected | Regenerated from backend output (content only). |
| AI-KNOWLEDGE-BOOTSTRAP.md | Regenerated via `go run ./cmd/goal ai`. |
| prd.json | US-042 passes:true. |
| progress.txt | Append entry. |

## Steps (ordered)
1. Add `-update-goldens` regeneration test (package corpus) using backend.Transpile.
2. Run it with `-update-goldens` to rewrite all exact goldens from AST output.
3. Switch the 4 exact-tier tests (TestTranspileRunner, TestDoctestRunner,
   TestCorpusTranspile, TestCorpusDoctest) to backend.Transpile.
4. Flip cmd/goal default engine to engineAST; update usage text; flip main_test.go.
5. Regenerate AI-KNOWLEDGE-BOOTSTRAP.md.
6. Run verifyCommands (build/vet/test) + confirm the US-041 behavioral gate green.

## Reuse / Patterns
- Transpiler interface seam: engine swap is a one-line change per test.
- `-update-*` flag pattern from internal/parser/snapshot_test.go.
- isDoctestSidecar (generate.go) classifies which output a golden represents.
- backend does NOT import corpus, so package-corpus tests may import backend
  (ast_gate_test.go precedent); pipeline_test (external) may import backend too.

## Risks
- A transpile error during regeneration: ruled out — US-041 ran all cases through
  backend at the behavioral tier with zero failures.
- Splice losing coverage: it retains the behavioral tier + CLI flag.
