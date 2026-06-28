# Plan — goal fmt source printer (US-045)

## Components

### 1. `internal/goalfmt/format.go` (new)

The formatter core. Position-driven reindenter; never prints the bare AST.

Interface contract:

```go
package goalfmt

// Source formats goal source: it normalizes leading indentation, trailing
// whitespace, and blank-line runs while preserving every comment and all
// intra-line content verbatim. It is idempotent: Source(Source(s)) == Source(s).
// Returns the parse error (and "" output) for source that does not parse.
func Source(src string) (string, error)
```

Internal helpers (unexported):

```go
func reindent(src string) string            // the whitespace normalizer
func isOpener(k token.Kind) bool             // ( [ {
func isCloser(k token.Kind) bool             // ) ] }
func collapseBlanks(lines []string) []string // drop leading/trailing + dedupe blank runs
```

Algorithm:
1. `parser.ParseFile(src)` — return err if it fails (validity gate).
2. `lexer.Tokens(src)` — walk tokens in offset order; maintain `depth` via
   opener/closer KINDS. Record per 1-based line: the depth at line start, whether
   the line's first token is a closer, and (defensively) continuation lines of any
   multi-line token (mark verbatim; mark the start line no-trim-right).
3. Rebuild from `strings.Split(src, "\n")`: for each non-protected line, trim
   leading whitespace, trim trailing whitespace, and if non-blank re-prefix with
   `depth` tabs (dedent by one when the first token is a closer). Protected
   continuation lines are emitted verbatim.
4. `collapseBlanks` then `strings.Join(..., "\n") + "\n"`.

### 2. `internal/goalfmt/format_test.go` (new, package goalfmt)

- `TestIdempotentOverCorpus`: load `../../corpus/manifest.json` via `corpus.Load`,
  enumerate unique inputs via `corpus.CaseInputs` (mirrors `TestParseGate`), read
  each from repo root `../..`, assert `Source(src)` succeeds and
  `Source(once) == once`. `t.Fatalf` if zero inputs.
- `TestPreservesComments`: a small inline sample with `//` and `///` comments;
  assert each comment substring is present in `Source(sample)`.
- `TestRejectsUnparseable`: assert `Source` returns an error on malformed input.

### 3. `cmd/goal/main.go` (edit)

- Add a `guideCommands` entry: `{Name:"fmt", Summary:"format .goal source (canonical layout, comment-preserving)", Usage:"goal fmt [-w] [path]", Flags:[{Name:"-w", Summary:"write result back to each file instead of printing to stdout"}]}`.
- Add `case "fmt":` in `run` → `parseFmtFlags(rest)` → `cmdFmt(path, write, out, errOut)`.
- `parseFmtFlags`: clone of `parseFixFlags` accepting `-w`/`--write` and one path.
- `cmdFmt`: clone the file-discovery shape of `cmdFix` (stat path; if dir, walk
  `*.goal`; else single file). For each file call `goalfmt.Source`; on `-w` write
  changed files back and list them; otherwise print formatted source to stdout.
  Parse/I/O errors are operational failures.

### 4. `cmd/goal` AI guide golden

Adding a subcommand changes `guideCommands`, so regenerate the checked-in bootstrap
golden: `go run ./cmd/goal ai > AI-KNOWLEDGE-BOOTSTRAP.md` (per the Codebase
Patterns note: `TestBootstrapGoldenMatches` fails otherwise).

## Integration Points

- `internal/goalfmt` imports `internal/parser`, `internal/lexer`, `internal/token`
  only — no cycle (none of those import goalfmt).
- The test imports `internal/corpus`; corpus does not import goalfmt, so a
  `package goalfmt` internal test importing corpus is safe.
- `cmd/goal` already imports nothing from goalfmt; add the import.

## Testing Strategy

- Unit + corpus tests in `internal/goalfmt` (above).
- `cmd/goal/main_test.go` already exercises subcommand dispatch and the bootstrap
  golden; regenerating the golden keeps `TestBootstrapGoldenMatches` green. Add a
  light `goal fmt` smoke test only if needed.

## Requirement traceability

- FR-1/FR-2 → `cmd/goal` `cmdFmt` + flags.
- FR-3 (comments) → `reindent` emits raw source lines; `TestPreservesComments`.
- FR-4 (idempotent) → `reindent` design; `TestIdempotentOverCorpus`.
- FR-5/FR-6 (meaning/layout-only) → only whitespace touched; `TestIdempotentOverCorpus`
  re-runs `Source` (which re-parses) so formatted output must still parse (AC-4).
- AC-3 → `Source` returns parse error; `TestRejectsUnparseable`.
