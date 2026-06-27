# Technical Requirements / Research — US-036

## Where the work lands

- `internal/backend/emit.go` — add enum-match lowering (`enumMatch`) and wire it
  into:
  - `matchStmt` default branch (statement position, when the qualifier names a
    resolved enum),
  - `returnStmt` (value position: `return match …`),
  - `stmt` `*ast.DeclStmt` case (value position: `var name T = match …`).
  Extend `selectorExpr` to export an enum payload-binding field access
  (`a.since` -> `v.Since`).
- `internal/backend/lower.go` — small helpers as needed (mirroring
  `internal/pass/match.go`).
- New corpus fixture: a value-position-match `.goal` + `.go.expected` under
  `testdata/` (or a feature examples dir), then regenerate `corpus/manifest.json`
  via `go run ./cmd/corpus-gen -root .`.

## Reference encoding (legacy splice)

`internal/pass/match.go` is the known-good enum-match lowering:
- `switch [v := ]scrut.(type) { case Enum_Variant: <arm> … default: panic(...) }`
- the guard `v :=` is emitted only when some arm references its binding;
- field accesses on the binding are exported;
- `_` rest arm becomes a real `default:`, else a panicking default with the
  message `unreachable: non-exhaustive <Enum> (compiler invariant violated)`.

## Existing AST-backend machinery to reuse

- `matchQualifier(m)` — first variant-pattern arm's enum name.
- `enumOf(info, name)` / `sema.Enum.FieldSet[variant]` — field-name sets.
- `e.renames` map + `armBodyRenamed` — binding rename to the gensym.
- `e.gensym("v")` — scope-aware temp name (no `__goal_` prefix; US-035).
- `usesIdent(body, name)` — whether an arm references its binding.

## Verification

- prd.json verifyCommands: `go build ./...`, `go vet ./...`,
  `go test ./... -count=1`.
- AC: a backend test runs all 02-match cases + the new value-position case
  through `backend.Transpile` + `corpus.RunCompile` (temp-module build+vet).
