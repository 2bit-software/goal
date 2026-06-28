# Implementation Tasks — US-011

## Task 1: token kinds, Pos, and round-trip API
**Status**: pending
**Files**: `internal/token/token.go`
**Depends on**: (none)
**Spec coverage**: FR-1 (kinds for every lexeme), FR-2 (contextual keywords are
IDENT), FR-3 (String + Lookup round-trip), FR-4 (Pos + ordering).
**Verify**: `go build ./internal/token/ && go vet ./internal/token/`

### Instructions
- Create package `token` at `internal/token/token.go` (import path `goal/internal/token`).
- `type Kind int` with `iota` constants, grouped go/token-style:
  - `ILLEGAL`, `EOF`, `COMMENT`, `DOC_COMMENT`.
  - `literal_beg`, `IDENT`, `INT`, `FLOAT`, `IMAG`, `CHAR`, `STRING`, `literal_end`.
  - `operator_beg` … operators/delimiters … `operator_end`. Include Go operators
    plus goal `QUESTION` (`?`), `FAT_ARROW` (`=>`), and `ELLIPSIS` (`...`).
  - `keyword_beg` … Go reserved words + goal `MATCH`, `ENUM`, `ASSERT` … `keyword_end`.
  - Do NOT add kinds for `implements`/`sealed`/`from`/`derive` (they stay IDENT).
- `var kindNames = [...]string{...}` indexed by Kind; `func (k Kind) String() string`
  returns the name or `token(N)` when out of range.
- `var keywords map[string]Kind` built in `init()` from the keyword range;
  `func IsKeyword(name string) bool`.
- `var operatorNames map[string]Kind` built from the operator range for punctuation
  round-trip.
- `func Lookup(name string) (Kind, bool)`: keyword map, then operator map; else
  `(ILLEGAL, false)`.
- `type Pos struct { Offset, Line, Col int }` (0-based Offset, 1-based Line/Col);
  `func (p Pos) Less(q Pos) bool { return p.Offset < q.Offset }`;
  `func (p Pos) String() string` → `"line:col"`.
- `type Token struct { Kind Kind; Lit string; Pos Pos }`.

## Task 2: unit tests
**Status**: pending
**Files**: `internal/token/token_test.go`
**Depends on**: Task 1
**Spec coverage**: the acceptance-criteria test (Kind/String round-trip; Pos ordering).
**Verify**: `go test ./internal/token/ -count=1`

### Instructions
- `package token`, stdlib `testing` only (NO testify — project constraint).
- `TestKindStringRoundTrip`: iterate keyword range and operator range; assert
  `Lookup(k.String())` returns exactly `k` and ok==true. Add explicit spot checks:
  `?`→QUESTION, `=>`→FAT_ARROW, `...`→ELLIPSIS, `match`→MATCH, `enum`→ENUM.
- `TestContextualKeywordsAreNotKeywords`: for implements/sealed/from/derive assert
  `IsKeyword` is false and `Lookup` returns `(ILLEGAL, false)`.
- `TestLookupUnknown`: `Lookup("definitelyNotAToken")` → `(ILLEGAL, false)`.
- `TestPosOrdering`: `Pos{Offset:1}.Less(Pos{Offset:2})` true; reverse false;
  reflexive `Pos{Offset:5}.Less(Pos{Offset:5})` false.

## Coverage check
- Files: `token.go` (Task 1), `token_test.go` (Task 2) — both plan files covered.
- FRs: FR-1..FR-4 covered by Task 1; acceptance test covered by Task 2.
