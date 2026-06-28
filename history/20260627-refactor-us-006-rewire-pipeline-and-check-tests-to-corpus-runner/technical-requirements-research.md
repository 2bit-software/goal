# Technical Requirements & Research — US-006

## Existing seams (already built by US-001..US-005)

- `internal/corpus` exposes whole-corpus runners driven by `corpus/manifest.json`:
  - `RunTranspile(root, Case, Transpiler)` — KindTranspile cases (51), gofmt-normalized
    compare with doctest-sidecar fallback.
  - `RunDoctest(root, Case, Transpiler)` — KindDoctest cases (4), strict sidecar compare.
  - `RunCheck(root, Case, Checker)` — KindCheck cases (50), inline `// want` markers.
- `corpus.Load(path)(Manifest, error)` loads the manifest; `Manifest.Cases` is `[]Case`.
- Adapters: `TranspilerFunc(pipeline.Transpile)` and `CheckerFunc(check.Analyze)`.

## Import-cycle constraint

`internal/corpus` imports `internal/pipeline` and `internal/check`. Therefore the
rewired test files CANNOT be internal (`package pipeline` / `package check`) test
files — that would form an import cycle. They must be EXTERNAL test packages
(`package pipeline_test`, `package check_test`), which Go compiles separately and
permits to import a package that imports the package under test.

## Repo-root depth

From `internal/pipeline` and `internal/check`, the repo root is `../..` (same as
`internal/corpus`, which uses `repoRoot = "../.."` and
`manifestPath = "../../corpus/manifest.json"`).

## Plan sketch

- Replace `pipeline_test.go` body with an external-package test that loads the
  manifest and runs every KindTranspile case via `corpus.RunTranspile` and every
  KindDoctest case via `corpus.RunDoctest` (through `pipeline.Transpile`). Drop
  the local `mustFormat` helper (only used here).
- Replace `check_test.go`'s `TestCases` (and its `parseWants`/`wantRe`/`runCase`
  helpers, only used here) with an external-package test that runs every KindCheck
  case via `corpus.RunCheck` (through `check.Analyze`). Preserve `TestRegistryRuns`
  smoke coverage (move it to the external package, calling `check.Analyze`).
- Keep "fail loudly on zero cases" so a mis-generated manifest can't masquerade
  as green.

## Verification

- `go build ./...`, `go vet ./...`, `go test ./... -count=1` (prd verifyCommands).
- `grep` for `features` / `filepath.Join("..","..","features"` in the two files
  returns nothing.
