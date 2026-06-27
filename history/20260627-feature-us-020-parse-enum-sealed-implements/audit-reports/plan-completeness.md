# Plan Completeness Audit — US-020

Every spec requirement traces to a plan element:

- FR-1 (enum) -> `parseEnumDecl`/`parseVariant`/`parsePayloadField` +
  `TestParseEnumDecl`.
- FR-2 (sealed interface) -> `parseSealedInterfaceDecl` + `TestParseSealedInterface`.
- FR-3 (struct implements) -> `parseImplementsClause` + `parseStructType` edit +
  `TestParseImplements`.
- Acceptance criteria (qualified implements, multi-field payload, build/vet/test)
  all have a corresponding test or verify command.

No scope creep: the plan touches only `internal/parser`. No CRITICAL/MAJOR.

## Assumptions

- Goal-specific parse methods live in a new `goal_decl.go` rather than growing
  `parser.go`, mirroring the `ast` package's `goal_decl.go` split.
- Tests read example `.goal` files from `../../features/...` (cwd = package dir),
  matching how other internal tests reference repo files.
