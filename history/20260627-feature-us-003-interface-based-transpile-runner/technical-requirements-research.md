# Technical Requirements & Research — US-003

## Stated technical requirements

- `internal/corpus` defines `Transpiler interface { Transpile(src string) (pipeline.Output, error) }`.
- A runner gofmt-normalizes both got and want before comparing.
- A test runs every transpile case in the manifest against `pipeline.Transpile`
  and all pass.

## Codebase findings

- `pipeline.Transpile(src string) (pipeline.Output, error)` is a free function,
  not a method. An adapter (`TranspilerFunc`) lets it satisfy the `Transpiler`
  interface without changing pipeline.
- No import cycle: `internal/pipeline` does not import `internal/corpus`, so
  corpus may import pipeline.
- The manifest (`corpus/manifest.json`) stores repo-root-relative, slash-form
  paths. Tests in `internal/corpus` run with cwd = that dir; repo root is
  `../..` (see the existing `repoRoot` const in generate_test.go). Manifest is at
  `../../corpus/manifest.json`.
- gofmt normalization: `go/format`.Source, mirroring pipeline_test.go's
  `mustFormat` helper.
- Doctest nuance: feature-11 example `.go.expected` files are doctest sidecars
  (`Output.Test`), while `testdata/doctest_funcs.go.expected` is the main Go
  output (`Output.Go`) even though its source also has `///` doctests. So the
  presence of doctests in the source does NOT determine which output the golden
  represents. The runner therefore passes a transpile case when the gofmt-
  normalized golden equals the produced `Output.Go` OR (when non-empty) the
  produced `Output.Test`.

## Design

- New file `internal/corpus/runner.go`:
  - `Transpiler` interface + `TranspilerFunc` adapter.
  - `RunTranspile(root string, c Case, tp Transpiler) error` — reads input and
    expected relative to root, transpiles, gofmt-normalizes both sides, compares
    (Go-or-sidecar), returns a descriptive error on any mismatch/read/parse
    failure.
- New file `internal/corpus/runner_test.go`:
  - Loads `../../corpus/manifest.json`, runs every `KindTranspile` case against
    `TranspilerFunc(pipeline.Transpile)` via subtests; fails if zero cases.
