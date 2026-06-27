# Verification — US-003 Interface-based transpile runner

## Acceptance criteria

- [x] `internal/corpus` defines `Transpiler interface{ Transpile(src string) (pipeline.Output, error) }`
      and a runner (`RunTranspile`) that gofmt-normalizes both got and want
      before comparing. (`internal/corpus/runner.go`)
- [x] A test runs every transpile case in the manifest against
      `pipeline.Transpile` and all pass. `TestTranspileRunner` ran 51 transpile
      subtests, all PASS. Fails loudly (`t.Fatalf`) if zero cases.

## Verify commands (prd.json verifyCommands)

- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS (all packages ok; corpus 51/51 transpile cases)

## Findings

No CRITICAL findings. No MAJOR findings.

### MINOR
- "Match Go or doctest sidecar" is a deliberate tolerance for the manifest's
  collapsing of doctest pairs into transpile pairs (documented in runner.go and
  business-spec FR-4). A dedicated doctest sidecar runner is US-005.

## Assumptions
- A transpile case passes when the golden equals the normalized main output or
  the normalized doctest sidecar; verified correct for all 51 cases, including
  feature-11 (sidecar golden) and testdata/doctest_funcs (Go-output golden).
- Repo root is `../..` and the manifest is `../../corpus/manifest.json` relative
  to `internal/corpus`, reusing the existing `repoRoot` const.
