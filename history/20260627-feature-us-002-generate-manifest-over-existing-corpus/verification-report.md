# Verification Report — US-002

## Project gates (prd.json verifyCommands)

- `go build ./...` — PASS (clean)
- `go vet ./...` — PASS (clean)
- `go test ./... -count=1` — PASS (all packages ok, including
  `goal/internal/corpus`)

## Acceptance criteria

- AC: "A generator walks features/NN/examples, testdata, and testdata/check and
  writes corpus/manifest.json without moving any source file" — PASS.
  `corpus.Generate` + `cmd/corpus-gen` wrote `corpus/manifest.json`; the only
  modified tracked source file is `internal/corpus/corpus.go` (added a
  `//go:generate` doc line). No corpus file moved or rewritten.
- AC: "A test asserts the generated manifest contains 51 transpile pairs and 50
  check cases" — PASS. `TestGenerateCounts` asserts exactly 51 transpile and 50
  check; on-disk `corpus/manifest.json` shows 51 `"kind": "transpile"` and 50
  `"kind": "check"` (101 cases total).

## Additional checks

- `TestGenerateDeterministic` — regeneration is byte-identical (diffable manifest).
- `TestGenerateNonDestructiveShape` — emitted case fields are well-formed.

## Result

All gates and acceptance criteria green. Recommend PASS.

## Assumptions

- Feature 11-doctests example pairs are indexed as transpile cases (they ship
  `.go.expected`), consistent with the prd 40+11=51 audit.
- Check cases carry empty `Expected`/`Normalize=none`; inline `// want` markers
  are honored by the later check runner (US-004).
