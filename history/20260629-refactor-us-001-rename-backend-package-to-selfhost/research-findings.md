# Research Findings — US-001

This is an internal rename following a well-established in-repo pattern (7 prior
package ports: token, lexer, ast, parser, sema, project, pipeline). No external
research required. Key facts confirmed against the codebase:

## Port harness (internal/selfhost/selfhost.go)
- `BuildTranspiled(layout map[string]*project.Package)` — COMPILE gate: transpiles
  every package in `layout` into a throwaway `module goal` temp dir keyed by
  module-relative dir, then `go build ./...`.
- `BuildAndTest(relDir, pkg, testFiles, deps)` — BEHAVIORAL gate: transpiles `deps`
  + `pkg`, copies each `testFiles` path verbatim into the temp package dir, runs
  `go test ./<relDir>`. Copied test files MUST compile in the temp module (only
  ported packages + stdlib are present) — so a test file importing
  `goal/internal/corpus` cannot be used.
- `discoverPorted(t, name)` helper in port_test.go discovers `../../selfhost/<name>`.

## backend dependency closure
- Direct in-module imports across the 6 non-test files: ast, parser, pipeline,
  sema, project, token. Transitive: lexer (via parser/sema/project/pipeline).
  Full layout = token, lexer, ast, parser, sema, project, pipeline, backend.
- Foreign imports pass through: fmt, go/format, go/importer, go/token, go/types,
  strconv, strings, unicode.

## Reserved-word collisions
- `grep` for bare `match`/`enum`/`assert` identifiers in the 6 non-test files: NONE.
  Every hit is string-literal content (e.g. `e.fail("enum ... not resolved")`) or a
  `token.MATCH`/`token.ENUM` constant. Verbatim copy needs zero source edits.

## Test subset
- backend_test.go is one monolithic file importing corpus/project and reading
  repo-relative fixtures. The 12 self-contained tests must be split into their own
  fixture-free file so BuildAndTest can copy a compilable suite. (Decision recorded
  in technical-requirements-research.md.)
