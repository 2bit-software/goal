# Tasks — US-036

- [ ] T1. Add `matchPos` enum (posStmt/posReturn/posVar) and the `enumMatch`
      helper in `internal/backend/emit.go` (type-switch, usesBinding guard,
      per-variant cases, rest/panicking default, position wrapping).
- [ ] T2. Add emitter fields `armBinding string` + `armFields map[string]bool`
      and the field-export branch in `selectorExpr`; set/restore them per arm
      (alongside the existing `renames` rename).
- [ ] T3. Wire statement dispatch: `matchStmt` default branch routes a
      resolved-enum match to `enumMatch(m, posStmt, "")`.
- [ ] T4. Wire return dispatch: `returnStmt` routes a single `*ast.MatchExpr`
      result over a resolved enum to `enumMatch(m, posReturn, "")`.
- [ ] T5. Wire var dispatch: `stmt` `*ast.DeclStmt` routes a
      `var name T = match` (single ValueSpec, single match value, resolved enum)
      to emit `var name T` then `enumMatch(m, posVar, name)`.
- [ ] T6. Add a new value-position-match corpus fixture
      (`features/02-*/examples/<name>.goal` + `.go.expected`, golden produced by
      the splice engine). Regenerate `corpus/manifest.json`.
- [ ] T7. Update `internal/corpus/generate_test.go` transpile count 51 -> 52.
- [ ] T8. Add `TestASTEngineEnumMatchBehavioralTier` in
      `internal/backend/backend_test.go` driving the four 02-match cases + the
      new case through `backend.Transpile` + `corpus.RunCompile`.
- [ ] T9. Run verifyCommands: `go build ./...`, `go vet ./...`,
      `go test ./... -count=1`. Fix until green.
