# Tasks — US-002

## T1: Implement `Generate` in internal/corpus
- **Files**: `internal/corpus/generate.go`, `internal/corpus/corpus.go` (go:generate doc)
- **Spec coverage**: FR-1, FR-2, FR-3, FR-5
- **Depends on**: none (foundation)
- Walk `features/*/examples/*.goal` + `testdata/*.goal` (paired with
  `.go.expected`) into transpile cases; walk `testdata/check/**/*.goal` into
  check cases. Sorted by Input, repo-relative slash paths. Read-only.

## T2: Add generator command
- **Files**: `cmd/corpus-gen/main.go`
- **Spec coverage**: FR-4
- **Depends on**: T1
- Calls `corpus.Generate(root)`, marshals indented JSON, writes
  `corpus/manifest.json`.

## T3: Generate the committed manifest
- **Files**: `corpus/manifest.json`
- **Spec coverage**: FR-4
- **Depends on**: T2
- Run the command to produce the on-disk manifest.

## T4: Tests
- **Files**: `internal/corpus/generate_test.go`
- **Spec coverage**: AC counts + determinism
- **Depends on**: T1
- Assert 51 transpile + 50 check; assert regeneration is byte-identical;
  stdlib testing only.

## Coverage check
- FR-1 → T1; FR-2 → T1; FR-3 → T1; FR-4 → T2,T3; FR-5 → T1,T4.
- Plan files: generate.go → T1; corpus.go → T1; corpus-gen/main.go → T2;
  manifest.json → T3; generate_test.go → T4.
