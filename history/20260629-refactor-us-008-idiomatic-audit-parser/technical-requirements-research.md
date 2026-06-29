# Technical requirements / research — US-008

## How the parser handles errors (key finding)
`selfhost/parser` uses an **error-accumulator** pattern, not per-function
`(T, error)` propagation:
- `type parser struct { ... errs []error ... }` accumulates errors.
- `(*parser).errorf(pos, format, args...)` appends to `p.errs`.
- Every internal parse helper returns just an AST node (or `nil`); none return
  `error`. Malformed input is recorded into `p.errs` and the parser keeps making
  progress (`expect`/`advance` always advance).
- The single `(T, error)` function is the exported entry point
  `ParseFile(src) (*ast.File, error)`, which joins `p.errs` at the end.

Consequence: there is **no internal `(T, error)` propagation surface** to convert
to `Result`/`?`. The one `(T, error)` function is exported and oracle-pinned.

## Switch survey
No `enum` declaration exists in `selfhost/parser`. All `switch` statements are
over:
- `p.kind()` / `tok` / `k` — `token.Kind`, which is `type Kind int` (deliberately
  NOT a goal `enum` per US-005), so not an enum scrutinee.
- type-switches `x.(type)` over `ast` category interfaces (`Node`/`Stmt`/`Expr`/…)
  — those cannot be sealed per US-007 (§9 switch-coexistence + oracle break).
- bare `switch { ... }` over boolean conditions.
There is therefore no `switch`-over-in-file-`enum` to convert to `match`.

## Machine check
`goal fix selfhost/parser/*.goal` produces no content diff. Its only stderr is a
result-sig SKIP for the exported `ParseFile` (non-propagating, exported — the
fixer deliberately refuses it), i.e. zero auto-convertible propagation sites.

## Reused gates
- `internal/selfhost` port gate (BuildTranspiled compile gate + BuildAndTest
  behavioral gate against `../parser/parser_test.go`) — runs under `task check`.
- `task fixpoint` — byte-identical goal-c-1/goal-c-2 over the whole selfhost tree.
