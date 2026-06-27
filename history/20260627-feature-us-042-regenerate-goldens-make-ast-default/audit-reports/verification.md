# Verification — US-042

## Verify commands (prd.json verifyCommands) — all green
- `go build ./...` — OK
- `go vet ./...` — OK
- `go test ./... -count=1` — OK (all packages pass)

## Acceptance criteria
- Default engine resolves to AST with no flag — TestParseFlags (cmd/goal): PASS.
- `--engine=splice`/`--engine=ast` accepted, unknown errors — TestParseFlagsEngine: PASS.
- Goldens equal AST backend output (gofmt-normalized) — regenerated via
  TestUpdateGoldens (-update-goldens); 52 goldens produced from backend output.
- Exact transpile tier green on AST — TestTranspileRunner / TestCorpusTranspile
  (52 cases): PASS.
- Exact doctest tier green on AST — TestDoctestRunner / TestCorpusDoctest: PASS.
- Whole-corpus behavioral gate still green — TestASTEngineWholeCorpusBehavioralGate
  (108 cases incl. package mode): PASS.
- Docs name AST default — AI-KNOWLEDGE-BOOTSTRAP.md regenerated; TestBootstrapGoldenMatches: PASS.
- Splice retained — `--engine=splice` branch intact; splice still exercised by the
  behavioral tier (RunCompile/RunDoctestExec/RunPackage via pipeline.Transpile).

## Regression found & fixed during implement
- Flipping the default broke TestBuildErrorMapsToGoalSource: the AST package path
  emitted no `//line` source map. Fixed by exporting pipeline.AddLineDirectives
  (name-based, engine-agnostic) and applying it in backend.TranspilePackage, and
  by routing the default CLI package path through backend.TranspilePackage
  (cross-file fact merge + single shared prelude) instead of the per-file helper.

## Result: PASS
