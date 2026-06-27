# Tasks — US-038

- [ ] **T1 — lower.go pure helpers** (`internal/backend/lower.go`)
  Add `baseType`, `zeroLit`, `zeroSafety` (mirrored from analyze.ZeroLit /
  pass.zeroSafety, reading a `decls` map + `sema.Info` enum/sealed sets),
  `needsFmtImport`, `importsPkg`, `presentFieldNames`, `structFieldsOf`.
  No dependencies.

- [ ] **T2 — emit.go assert + defaults dispatch** (`internal/backend/emit.go`)
  Depends on T1. Add `typeDecls` field; `exprText`, `buildTypeDecls`; wire
  `e.typeDecls` in `emitFile`; `file()` fmt-import injection; `stmt()` AssertStmt
  case + `assertStmt`; route CompositeLit through `compositeLit` + `defaultEntries`.

- [ ] **T3 — tests** (`internal/backend/backend_test.go`)
  Depends on T2. Add `TestASTEngineDefaultsAssertBehavioralTier` (6 cases via
  corpus.RunCompile) and `TestASTEngineDefaultsAssertEncoding` (shape pins).

- [ ] **T4 — verify** Run prd.json verifyCommands (`go build ./...`,
  `go vet ./...`, `go test ./... -count=1`); fix any red before finishing.
