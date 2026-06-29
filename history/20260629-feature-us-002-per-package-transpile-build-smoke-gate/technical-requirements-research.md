# Technical Requirements / Research — US-002

## Hints from the story

- The compiler source is already valid goal (a Go superset), so the gate can
  feed `internal/<pkg>/*.go` (treated as goal source) through the goal package
  driver `backend.TranspilePackage` and then `go build` the result.
- `go build` is the real gate because the checker stays silent on miscompiles
  (that is how US-001 was found).
- This is the verification harness every later port story reuses.

## Existing patterns to reuse

- `internal/corpus/package_runner.go` already shows the canonical
  transpile-then-build flow: build a `project.Package` from files, call
  `backend.TranspilePackage`, assert each generated file is valid Go via
  `go/format`, then write the generated files plus their foreign imports into a
  throwaway temp module and run `go build ./...`. `moduleName`, `copyGoFiles`,
  and the temp-module wiring are directly relevant.
- `cmd/goal/main.go` shows the `-overlay` approach: transpile packages and map
  the real source paths to temp files so inter-package imports resolve in the
  real module. This is attractive for the compiler packages because they import
  each other (lexer->token, parser->token/lexer/ast, ...): overlaying the
  generated Go over the real package dirs and running `go build ./internal/...`
  lets the module resolve the import graph without hand-wiring each dependency.

## Open considerations

- Package files: `project.Package` expects `project.File{Path, Name, Src}`.
  Read the non-`_test.go` `*.go` files in each `internal/<pkg>` dir as the
  package's goal source.
- `backend.TranspilePackage` returns `pipeline.PackageOutput{Files, Tests}`.
  Only `Files` are needed for the build; `_test.go` originals are ignored by
  `go build`.
- The overlay must replace every original `.go` file of a covered package so the
  build does not see duplicate declarations from both the original and the
  generated file. Generated file names mirror source file names (see
  `writeOverlay` in cmd/goal/main.go), plus any extra prelude file the backend
  emits — overlay/temp-module placement must account for that.
- Skip `ast/dump.go` reflect concerns are out of scope here (that is US-007);
  this gate just builds whatever the front-end currently emits for each package.
