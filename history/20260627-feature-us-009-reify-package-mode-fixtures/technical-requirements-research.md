# Technical Requirements / Research — US-009

## Existing shape

- `internal/corpus` defines `Case{ID,Kind,Input,Expected,Mode,Normalize}` and a
  JSON `Manifest`; `Generate(root)` walks `features/*/examples`, `testdata/*`,
  and `testdata/check/**` into cases. `cmd/corpus-gen` writes
  `corpus/manifest.json`.
- `pipeline.TranspilePackage(*project.Package) (PackageOutput, error)` lowers a
  multi-file package against merged tables and emits one shared
  `goal_prelude.go`. Foreign types are enriched via
  `analyze.EnrichForeign(tables, srcs, pkg.Dir, nil)` → `DefaultResolver`, which
  walks up from `pkg.Dir` to the module `go.mod` (`module goal`) and resolves
  in-module import paths (e.g. `goal/internal/pipeline/testdata/extpkg`).
- Existing inline package tests:
  - `pipeline_package_test.go`: cross-file `demo` package (math.goal + types.goal),
    no imports — proves cross-file enum match + closed-E Result with one prelude.
  - `foreign_test.go`: `conv` package importing
    `goal/internal/pipeline/testdata/extpkg`, exercising `derive func` over a
    foreign struct.

## Plan

- Extend `corpus.Case` with an optional `Package *PackageSpec` carrying
  `Name`, `Files` (repo-relative `.goal` paths), and `Imports`
  (import-path → repo-relative dir of the foreign Go package = the declared
  import map). Package cases are `Kind=transpile`, `Mode=package`, with
  `Input` = the fixture directory (kept non-empty/meaningful).
- Place fixtures under `testdata/package/<name>/` (cross-file-demo,
  foreign-derive) plus a `pkg.json` descriptor declaring `name` + `imports`.
- `Generate` discovers `testdata/package/*/pkg.json`, globs the dir's `.goal`
  files, and emits one `Mode=package` case each.
- Add `PackageTranspiler` seam + `PackageTranspilerFunc` adapter
  (over `pipeline.TranspilePackage`) and `RunPackage(root, Case, pt)` that builds
  a `project.Package` (Dir = fixture dir, so DefaultResolver resolves in-module
  imports), transpiles, asserts every generated file is valid Go, and compiles
  the whole package — wiring each declared foreign import into a temp module —
  via `go build ./...`.
- Regenerate `corpus/manifest.json`; update the generate-count test to count
  file-mode vs package-mode transpile cases; update the shape test to not
  require `Expected` for package cases.

## Risks

- Foreign-derive generated Go must actually compile (the old inline test only
  string-asserted). If the derive body is not self-contained, fall back to
  transpile-success + valid-Go as the pass criterion.
