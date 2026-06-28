# Implementation Tasks — US-012 Lex core Go tokens with positions

## Task 1: Implement internal/lexer
**Status**: completed
**Files**: `internal/lexer/lexer.go`
**Depends on**: (none — builds on existing `internal/token`)
**Spec coverage**: FR-1, FR-2, FR-3, FR-4, Error Handling (ILLEGAL)
**Verify**: `go build ./internal/lexer/ && go vet ./internal/lexer/`

### Instructions
Create package `lexer` importing `goal/internal/token`, `unicode`, and
`unicode/utf8`.

- `Lexer` struct: `src string`, `offset int` (start of `ch`), `rdOffset int`
  (next byte), `ch rune` (current rune, -1 at EOF), `line int`, `lineStart int`
  (byte offset of current line's first byte).
- `New(src string) *Lexer`: init, prime first rune via `next()`, set `line = 1`.
- `next()`: decode the rune at `rdOffset` with `utf8.DecodeRuneInString`; when
  the previous `ch` was `'\n'`, increment `line` and set `lineStart = offset`.
  Advance `offset`/`rdOffset`. Set `ch = -1` at end.
- `pos()`: build `token.Pos{Offset: offset, Line: line, Col: offset - lineStart + 1}`.
- `Next() token.Token`:
  - skip spaces/tabs/`\r`/`\n` (newlines update line via `next()`).
  - capture start pos.
  - letter/`_` start → scan identifier; classify with `token.Lookup` (keyword)
    else `token.IDENT`. Lit = the word.
  - digit start (or `.` followed by digit) → scan number → INT/FLOAT/IMAG.
  - `"` → string, `` ` `` → raw string (both STRING); `'` → CHAR. Handle `\`
    escapes inside `"`/`'`; raw strings run to the next backtick.
  - `/` then `/` → line COMMENT (to end of line, not consuming `\n`); `/` then
    `*` → block COMMENT (to `*/`). NOTE: do NOT special-case `///` here — that
    is US-013; a `///` simply scans as a `//` line comment for now.
  - otherwise operator/delimiter via a `switch` with longest-match nested
    lookahead, mapping to the token Kinds. Keep `?`, `=>` (treat `=` `>` as
    ASSIGN then GTR), and `...` OUT of scope — but DO handle `.` (PERIOD) and
    `..`-less forms normally; a bare `..`/`...` is not in any US-012 sample.
  - EOF → `token.Token{Kind: token.EOF, Pos: ...}`.
  - unrecognized byte → `token.ILLEGAL` token with Lit = the rune's text; still
    advance so the lexer makes progress.
- `Tokens(src string) []token.Token`: loop `New(src).Next()` appending until and
  including the EOF token; return the slice.

Reference: model the cursor/`next()`/scan-method shape on Go's
`src/go/scanner/scanner.go`, but emit our `token.Token` and compute Pos directly.
Keep it stdlib-only (project is zero-dependency).

---

## Task 2: Lexer tests
**Status**: completed
**Files**: `internal/lexer/lexer_test.go`
**Depends on**: Task 1
**Spec coverage**: all acceptance criteria (multi-line position test is the AC)
**Verify**: `go test ./internal/lexer/ -count=1`

### Instructions
Package `lexer` (internal test), stdlib `testing` only — NO testify.

- `TestTokenizeMultiLineSample`: a fixed multi-line source (e.g. a small `func`
  with a string, a number, an assignment, a return across several lines).
  Build the expected `[]token.Token` by hand with exact Kind, Lit (for
  idents/literals/keywords where meaningful), and full `token.Pos`
  (Offset, Line, Col). Assert length and element-by-element equality; assert the
  final token is `token.EOF`. This exercises line increment and column reset.
- `TestKeywordVsIdent`: `func`,`for`,`return` → keyword kinds; `implements`,
  `sealed`,`from`,`derive`,`foo` → `token.IDENT`.
- `TestLiterals`: `42`→INT, `3.14`→FLOAT, `1i`→IMAG, `'a'`→CHAR, `"hi"`→STRING,
  raw backtick string → STRING.
- `TestLongestMatchOperators`: inputs like `:=`, `<<=`, `&^`, `==`, `<=` each
  yield exactly one token of the expected multi-char Kind.
- `TestIllegal`: an unexpected byte (e.g. `#`) yields a `token.ILLEGAL` token and
  does not panic; lexing continues to EOF.

Then run the full project gates: `go build ./...`, `go vet ./...`,
`go test ./... -count=1`.
