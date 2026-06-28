# Implementation Plan — US-011 token package with positions

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/token/token.go` | `Kind` named type + constants for every goal lexeme; `String()`; `Lookup(name)`; `Pos{Offset,Line,Col}` + `Less`; minimal `Token{Kind,Lit,Pos}`. |
| `internal/token/token_test.go` | stdlib-only unit tests: Kind/String round-trip; Pos ordering. |

### Modified Files
None. Self-contained new package; no existing code changes (the splice front-end is
untouched per the prd phasing).

## Design

Module path is `goal`, so the package import path is `goal/internal/token`.

### Kind
- `type Kind int` with `iota` constants:
  - Sentinels: `ILLEGAL`, `EOF`.
  - `IDENT`, literals `INT`, `FLOAT`, `IMAG`, `CHAR`, `STRING`.
  - `COMMENT` (ordinary `//` and `/* */`), `DOC_COMMENT` (goal `///` trivia).
  - Operators/delimiters mirroring `go/token` (ADD, SUB, MUL, QUO, REM, AND, OR,
    XOR, SHL, SHR, AND_NOT, ADD_ASSIGN…, LAND, LOR, ARROW, INC, DEC, EQL, LSS, GTR,
    ASSIGN, NOT, NEQ, LEQ, GEQ, DEFINE, ELLIPSIS, LPAREN, LBRACK, LBRACE, COMMA,
    PERIOD, RPAREN, RBRACK, RBRACE, SEMICOLON, COLON).
  - Goal-specific: `QUESTION` (`?`), `FAT_ARROW` (`=>`). (`...` is `ELLIPSIS`,
    shared with Go variadic; `///` is `DOC_COMMENT`.)
  - Keywords: Go reserved words (break, case, chan, const, continue, default, defer,
    else, fallthrough, for, func, go, goto, if, import, interface, map, package,
    range, return, select, struct, switch, type, var) + goal reserved
    (`match`, `enum`, `assert`). Delimited by `keyword_beg`/`keyword_end` internal
    markers so the keyword range is iterable.
  - NOT keywords (remain IDENT): `implements`, `sealed`, `from`, `derive` — contextual.
- `var kindNames [...]string` indexed by Kind; `String()` returns the name (or
  `token(N)` for out-of-range, like go/token).
- `var keywords map[string]Kind` built once from the keyword range.
- `Lookup(name string) (Kind, bool)` — first checks the keyword map, then a
  punctuation/operator name→Kind map built from the same names table over the
  operator range; returns `(ILLEGAL, false)` for unknown. `IsKeyword(name)` helper.

### Pos
- `type Pos struct { Offset, Line, Col int }` — 0-based Offset, 1-based Line/Col.
- `func (p Pos) Less(q Pos) bool { return p.Offset < q.Offset }` — canonical order.
- `func (p Pos) String() string` — `"line:col"` for diagnostics.

### Token
- `type Token struct { Kind Kind; Lit string; Pos Pos }` — minimal carrier the lexer
  (US-012) will emit. Not required by the gate but cheap and forward-looking.

## Test Plan
- `TestKindStringRoundTrip`: for every Kind in the operator range and keyword range,
  `Lookup(k.String())` returns that exact `k`. Assert a few representative spellings
  (`?`→QUESTION, `=>`→FAT_ARROW, `...`→ELLIPSIS, `match`→MATCH) explicitly.
- `TestContextualKeywordsAreNotKeywords`: `IsKeyword` is false for implements/sealed/
  from/derive and `Lookup` returns `(ILLEGAL,false)` for them.
- `TestLookupUnknown`: `Lookup("nope")` → `(ILLEGAL,false)`.
- `TestPosOrdering`: earlier offset `Less` later; equal/greater are not; reflexive
  `Less` is false.

## Verification
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` (all green).
- `go test ./internal/token/ -run . -v` passes.

## Risk / Rollback
Additive new package; nothing imports it yet, so build risk is isolated to the package
itself. Rollback = delete the two files.
