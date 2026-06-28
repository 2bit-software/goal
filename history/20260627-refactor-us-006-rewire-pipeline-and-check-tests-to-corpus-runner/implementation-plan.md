# Implementation Plan — US-006 Rewire existing harnesses to runner

## File Inventory

### New Files
None.

### Modified Files
| File | Change |
|------|--------|
| `internal/pipeline/pipeline_test.go` | Rewrite as external `package pipeline_test`; drive every KindTranspile case via `corpus.RunTranspile` and every KindDoctest case via `corpus.RunDoctest`, both through `corpus.TranspilerFunc(pipeline.Transpile)`. Remove the hardcoded testdata/feature/feature-11 globs and the local `mustFormat` helper. |
| `internal/check/check_test.go` | Rewrite as external `package check_test`; drive every KindCheck case via `corpus.RunCheck` through `corpus.CheckerFunc(check.Analyze)`. Remove the hardcoded `testdata/check` walk and the local `wantRe`/`parseWants`/`runCase` helpers. Preserve the `TestRegistryRuns` spine smoke test (calling `check.Analyze`). |

## Reused Seams (already built)
- `corpus.Load(path) (Manifest, error)` and `Manifest.Cases []Case`.
- `corpus.RunTranspile(root, Case, Transpiler)`, `corpus.RunDoctest(root, Case, Transpiler)`,
  `corpus.RunCheck(root, Case, Checker)`.
- `corpus.TranspilerFunc(pipeline.Transpile)`, `corpus.CheckerFunc(check.Analyze)`.
- Kind constants `corpus.KindTranspile`, `corpus.KindDoctest`, `corpus.KindCheck`.

## Key Constraints
1. Import cycle: `corpus` imports `pipeline` and `check`. The rewired tests MUST be
   external `_test` packages (`pipeline_test`, `check_test`) so they can import
   `corpus`. Other `_test.go` files in those dirs remain internal-package tests —
   Go permits both packages in one directory.
2. Paths: from `internal/pipeline` and `internal/check`, repo root is `../..` and
   the manifest is `../../corpus/manifest.json` (same depth the corpus tests use).
3. Loud zero-case guard: each rewired test fails (`t.Fatalf`) if it drives zero
   cases of its kind.

## Step-by-step
1. Rewrite `internal/check/check_test.go`:
   - `package check_test`, import `testing` + `goal/internal/check` + `goal/internal/corpus`.
   - Constants `manifestPath = "../../corpus/manifest.json"`, `repoRoot = "../.."`.
   - `TestCorpusCheck`: Load manifest; for each `KindCheck` case, `t.Run(c.ID, ...)`
     calling `corpus.RunCheck(repoRoot, c, corpus.CheckerFunc(check.Analyze))`;
     `t.Fatalf` if zero ran.
   - Keep `TestRegistryRuns` (calls `check.Analyze`).
2. Rewrite `internal/pipeline/pipeline_test.go`:
   - `package pipeline_test`, import `testing` + `goal/internal/pipeline` + `goal/internal/corpus`.
   - Constants `manifestPath`, `repoRoot` as above.
   - `TestCorpusTranspile`: each `KindTranspile` case via `corpus.RunTranspile`; loud zero guard.
   - `TestCorpusDoctest`: each `KindDoctest` case via `corpus.RunDoctest`; loud zero guard.
3. Verify: `go build ./...`, `go vet ./...`, `go test ./... -count=1`; grep the two
   files for `features` and the join pattern (expect nothing).

## Risks
- A second package decl in a dir with existing internal `_test.go` files is legal,
  but a name collision between an external-package symbol and an internal one would
  fail to compile. Mitigation: the external files define only test funcs + unexported
  consts local to the external package; no collisions expected. The build/vet/test
  gate catches any.
