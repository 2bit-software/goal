# Tasks — goal fmt source printer (US-045)

Ordered by dependency; each task depends only on completed prior tasks.

## Task 1 — Formatter core (`internal/goalfmt/format.go`)

- Create package `goalfmt` with `Source(src string) (string, error)` and the
  unexported `reindent`, `isOpener`, `isCloser`, `collapseBlanks` helpers.
- Parse gate via `parser.ParseFile`; reindent via `lexer.Tokens` token-depth.
- Spec coverage: FR-3, FR-4, FR-5, FR-6, AC-2, AC-4 (the engine).
- Files: `internal/goalfmt/format.go`.
- Depends on: nothing.

## Task 2 — Tests (`internal/goalfmt/format_test.go`)

- `TestIdempotentOverCorpus` (AC-1, AC-4), `TestPreservesComments` (AC-2),
  `TestRejectsUnparseable` (AC-3).
- Files: `internal/goalfmt/format_test.go`.
- Depends on: Task 1.

## Task 3 — CLI subcommand (`cmd/goal/main.go`)

- Register `fmt` in `guideCommands`; add `case "fmt"`, `parseFmtFlags`, `cmdFmt`.
- Spec coverage: FR-1, FR-2, AC-3 (command surface + error path).
- Files: `cmd/goal/main.go`.
- Depends on: Task 1.

## Task 4 — Regenerate AI guide golden

- `go run ./cmd/goal ai > AI-KNOWLEDGE-BOOTSTRAP.md` so `TestBootstrapGoldenMatches`
  stays green after the new subcommand.
- Files: `AI-KNOWLEDGE-BOOTSTRAP.md`.
- Depends on: Task 3.

## Task 5 — Verify

- `go build ./...`, `go vet ./...`, `go test ./... -count=1` all green.
- Depends on: Tasks 1-4.

## Coverage matrix

| Requirement | Task |
|-------------|------|
| FR-1, FR-2  | 3    |
| FR-3, FR-6  | 1    |
| FR-4 / AC-1 | 1, 2 |
| FR-5 / AC-4 | 1, 2 |
| AC-2        | 1, 2 |
| AC-3        | 1, 2, 3 |
| golden drift | 4   |
