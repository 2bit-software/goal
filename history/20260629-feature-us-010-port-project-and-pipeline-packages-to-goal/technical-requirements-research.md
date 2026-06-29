# Technical Requirements / Research — US-010

## Port pattern (from prior self-host stories US-005..US-009)

Port = copy internal/<pkg>/*.go -> selfhost/<pkg>/*.goal (Go superset = valid
goal). Grep for bare `match`/`enum`/`assert` identifier collisions first
(string-literal/comment hits are fine). Then add a port_test that Discovers the
package and runs both gates:

- `selfhost.BuildTranspiled(layout)` — COMPILE gate (transpile + `go build` in a
  temp `module goal`). The layout must list every in-module dep package.
- `selfhost.BuildAndTest(relDir, pkg, testFiles, deps)` — BEHAVIORAL gate
  (transpile + copy the EXISTING internal/<pkg> test files beside the generated
  Go + `go test`). Pass ported deps in `deps map[string]*project.Package` keyed
  by module-relative dir.

## Dependencies

- `internal/project` imports `goal/internal/parser` (+ stdlib fmt, io/fs, os,
  path/filepath, sort, strings). parser pulls in lexer/ast/token. So deps =
  token, lexer, ast, parser.
- `internal/pipeline` (sourcemap.go) imports `goal/internal/ast`,
  `goal/internal/parser`, `goal/internal/token` (+ stdlib fmt, strings).
  pipeline.go is pure types, no imports. Deps = token, lexer, ast, parser.

## Reserved-word scan

No bare match/enum/assert identifier collisions in project.go, pipeline.go,
sourcemap.go (only comment hits). Verbatim copy, no edits.

## Test-suite selection

- `internal/project/project_test.go` — package `project`, stdlib-only
  (os/path/filepath/testing), creates temp dirs, calls Discover. SELF-CONTAINED
  -> INCLUDE.
- `internal/pipeline/sourcemap_test.go` — package `pipeline` white-box,
  strings/testing only. SELF-CONTAINED -> INCLUDE.
- `internal/pipeline/pipeline_test.go` — package `pipeline_test`, imports
  `goal/internal/backend` + `goal/internal/corpus`, reads `../../corpus/manifest.json`
  and repo-relative paths. NOT self-contained -> EXCLUDE (same spirit as
  US-007/US-008/US-009 fixture-dependent suite exclusions).

## Fixpoint

selfhost/ is auto-discovered by `task fixpoint` (project.Discover walks the
tree), so adding selfhost/project and selfhost/pipeline needs no harness change.
