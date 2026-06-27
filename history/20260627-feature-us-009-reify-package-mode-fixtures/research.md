# Research — US-009

Self-contained internal task; no external/web research required. Key findings
from reading the codebase:

- `corpus.Case` has `Mode` (file|package) already but no carrier for multiple
  files / an import map — needs a `Package *PackageSpec` extension.
- `pipeline.TranspilePackage(*project.Package)` is the package transpile seam
  (returns `PackageOutput{Files, Tests []GoFile}`, emitting one shared
  `goal_prelude.go`). corpus already imports pipeline; importing
  `internal/project` adds no cycle (project imports only scan).
- Foreign resolution: `analyze.DefaultResolver(importPath, fromDir)` walks up
  from `fromDir` to the nearest `go.mod` (module `goal`) and maps an in-module
  import path to its on-disk dir. So a fixture dir anywhere under the repo
  resolves `goal/internal/pipeline/testdata/extpkg` correctly with no special
  resolveDir. The existing foreign fixture package `extpkg` (types.go) stays as
  the foreign Go source.
- `Generate` count test asserts 51 transpile / 50 check / 4 doctest and
  `other == 0`; shape test requires every case have non-empty `Input` and every
  transpile case non-empty `Expected`. Package cases (Kind=transpile,
  Mode=package, empty Expected) require updating both tests: count file-mode vs
  package-mode, and exempt package cases from the Expected check (Input stays
  non-empty = fixture dir).
- The original `pipeline_package_test.go` compiled its output
  (`assertPackageCompiles`), so a temp-module `go build` is a proven pass bar
  for the cross-file case. The foreign case only string-asserted; its generated
  Go must be verified to compile — wire the declared foreign import into the
  temp module under its import-path tail (module `goal`).

Confidence: High (all derived from in-repo source).
