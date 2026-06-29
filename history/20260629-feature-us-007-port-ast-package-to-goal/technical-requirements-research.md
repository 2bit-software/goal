# Technical Requirements / Research — US-007

## Port pattern (established by US-005 token, US-006 lexer)

- Port = copy `internal/ast/*.go` -> `selfhost/ast/*.goal` (Go superset is valid
  goal). Grep for bare reserved words `match`/`enum`/`assert` first; only the
  bare standalone word collides (`.Enum`, `enumOf`, strings are fine).
- Add a `port_test.go` case in `internal/selfhost` that Discovers the package and
  runs BOTH gates:
  - `selfhost.BuildTranspiled(layout)` — compile gate (transpile + `go build` in
    a temp `module goal`); layout lists ast AND its token dep.
  - `selfhost.BuildAndTest("internal/ast", pkg, []string{"../ast/ast_test.go"}, deps)`
    — behavioral gate; deps carries token keyed by "internal/token".

## ast-specific findings

- ast depends only on `internal/token` (same dep shape as lexer).
- Reserved-word grep over internal/ast non-test files: the only hit is the
  string literal `len("assert")` in goal_stmt.go — a string, not an identifier,
  so it ports verbatim with no rename.
- `dump.go` is reflection-driven (Sexpr debug renderer) and per the prd notes is
  dropped from the self-hosted build. Confirmed nothing outside dump.go
  references `Sexpr`/`sexprDumper`, and `ast_test.go` does not reference
  Sexpr/Dump/reflect — so dropping dump.go is safe and the existing tests pass.
- Files to port: ast.go, walk.go, goal_decl.go, goal_expr.go, goal_stmt.go.
