# Technical Requirements & Research — US-008

## Established port pattern (from progress.txt Codebase Patterns + prior ports US-005/006/007)

Port a package by copying internal/<pkg>/*.go to selfhost/<pkg>/*.goal (Go superset
= valid goal). The Go-superset source is valid goal as-is; grep for the bare
reserved words `match`/`enum`/`assert` used AS identifiers first (here: none —
all occurrences are comments or token.MATCH/token.ENUM constants).

Two-gate verification lives in internal/selfhost (package selfhost_test,
internal/selfhost/port_test.go):

- `selfhost.BuildTranspiled(layout)` — COMPILE gate: transpile every package in
  the layout and `go build ./...` in a temp `module goal`. Layout keyed by
  module-relative dir ("internal/token", "internal/lexer", "internal/ast",
  "internal/parser").
- `selfhost.BuildAndTest(relDir, pkg, testFiles, deps)` — BEHAVIORAL gate:
  transpile the package + its in-module deps, copy the EXISTING white-box test
  files beside the generated Go, and `go test`. deps keyed by module-relative dir.

## Files to port (non-test, package parser)

internal/parser/{parser,goal_construct,goal_decl,goal_match,goal_stmt}.go →
selfhost/parser/*.goal. Imports: goal/internal/{ast,lexer,token} (all ported) plus
stdlib errors/fmt (pass through). No references to ast.Sexpr/dump (dropped from the
ported ast in US-007).

## Behavioral test selection

parser_test.go is fully self-contained (no shared helpers, no external file reads)
— it is the behavioral gate fed to BuildAndTest. The other suites
(goal_construct_test, goal_decl_test, goal_match_test, goal_stmt_test) read
repo-relative `../../features/...` example fixtures via the readExample/parseExample
helpers, and snapshot_test.go depends on ast.Sexpr (intentionally dropped from the
ported ast) plus repo-relative feature files + testdata/snapshots — none of these
are self-contained in the harness's throwaway temp module, so they are excluded
from the behavioral gate (same spirit as US-007 dropping the Sexpr path).

## Deps for the parser gate

token, lexer, ast (all discovered from selfhost/, transpiled into the temp module).
