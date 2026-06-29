# Technical Requirements / Research — US-006

## Established port pattern (from US-005, progress.txt)

- Port = copy `internal/<pkg>/*.go` -> `selfhost/<pkg>/*.goal` (Go superset is
  valid goal). Grep for bare `match`/`enum`/`assert` identifiers first and rename
  if found.
- Add a `port_test` in `internal/selfhost` that `project.Discover`s the new
  package and runs BOTH gates:
  - `selfhost.BuildTranspiled(layout)` — compile gate (transpile + `go build` in
    a temp `module goal`).
  - `selfhost.BuildAndTest(relDir, pkg, testFiles)` — behavioral gate (transpile
    + copy existing white-box tests + `go test`).

## Lexer-specific considerations

- `internal/lexer/lexer.go` imports `unicode`, `unicode/utf8` (pass through) and
  `goal/internal/token` (the ported package). So the temp module must contain the
  transpiled token package too.
- US-005 learning: "lexer (US-006) imports token, so its port_test layout must
  include BOTH selfhost/token and selfhost/lexer in the temp module
  (BuildTranspiled takes a multi-entry layout; extend BuildAndTest or pre-build
  deps if a ported package gains an internal dep)."
- BuildAndTest currently transpiles only ONE package into the temp module. For
  lexer it must also place the transpiled token package so the lexer's
  `goal/internal/token` import resolves and `go test ./internal/lexer` compiles.
  This requires extending BuildAndTest to accept dependency packages.

## Reserved-word check

- Will grep selfhost/lexer source for bare `match`/`enum`/`assert` before
  transpiling. lexer.go uses none expected (it has `New`, `Next`, `Tokens`,
  scan* helpers).
