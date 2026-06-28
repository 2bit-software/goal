# Technical Requirements / Research — US-002

## Existing seams (internal/backend/emit.go)

- Enum value-position match already lowers via `enumMatch(m, pos, name)` with
  `posReturn` (wired in `returnStmt`) and `posVar` (wired in `tryVarMatch` /
  `tryAssignMatch`). Arm bodies are wrapped by `armWrap` (bare / `return <body>` /
  `name = <body>`).
- Statement-position Result/Option match already lowers via `matchStmt` ->
  `resultMatch` / `closedResultMatch` / `optionMatch`, all emitting arm bodies via
  `armBody` (posStmt-equivalent).

## Gap

`resultMatch`/`closedResultMatch`/`optionMatch` are hard-wired to statement
position. A value-position Result/Option match (return or assignment RHS) reaches
the generic expr path -> `unsupported expression *ast.MatchExpr`.

## Approach

Generalize `resultMatch`/`closedResultMatch`/`optionMatch` to take `(m, pos,
name)` and emit arm bodies through `armWrap` (renamed arms through a new
`armBodyRenamedWrap`). Then dispatch:

- `returnStmt`: extend the existing single-result MatchExpr check to also route a
  Result/Option qualifier to `resultMatch`/`optionMatch` with `posReturn`.
- `tryVarMatch`: route a Result/Option match value to `resultMatch`/`optionMatch`
  with `posVar` (after emitting the explicit `var name T`).
- `tryAssignMatch`: route a Result/Option match RHS the same way, reusing
  `inferMatchType` for the `var name T` type (consistent with the enum path; a
  non-inferable arm set keeps the located deferral).

No new dependency; mirrors the enum value-position path.
