# Tasks — US-020 Parse enum, sealed, implements

Status: Task 1–4 completed.


## Task 1 — Add goal closed-type parse methods
**Files**: `internal/parser/goal_decl.go` (new)
**Depends on**: none (uses existing `ast`/`token`/`parser.go` helpers)
**Spec coverage**: FR-1, FR-2, FR-3
- Implement `parseEnumDecl`, `parseVariant`, `parsePayloadField`,
  `parseSealedInterfaceDecl`, `parseImplementsClause`, and an
  `isContextual` helper.

## Task 2 — Wire dispatch and struct implements
**Files**: `internal/parser/parser.go` (modified)
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-2, FR-3
- `parseDecl`: dispatch `token.ENUM` and the `sealed` contextual keyword.
- `parseStructType`: consume an optional implements clause into
  `StructType.Implements`.

## Task 3 — Corpus-driven structural tests
**Files**: `internal/parser/goal_decl_test.go` (new)
**Depends on**: Task 1, Task 2
**Spec coverage**: all acceptance criteria
- `TestParseEnumDecl`, `TestParseSealedInterface`, `TestParseImplements` over the
  `features/01-enums` and `features/07-implements` example inputs; assert no
  parse error and the variant/field/implements structure.

## Task 4 — Verify
**Depends on**: Task 1–3
**Spec coverage**: build/vet/test acceptance criterion
- Run `go build ./...`, `go vet ./...`, `go test ./... -count=1`; all green.
