# Verify: Acceptance Coverage — US-012

Full suite: `go build ./...`, `go vet ./...`, `go test ./... -count=1` all green
(every package ok, including the new `goal/internal/lexer`).

## Criterion → evidence

| Acceptance criterion | Evidence | Result |
|----------------------|----------|--------|
| Lexer tokenizes idents, literals, operators, delimiters with accurate Offset/Line/Col | `internal/lexer/lexer.go` (scanIdentifier/scanNumber/scanString/scanRawString/scanRune/scanOperator + pos()); `TestTokenizeMultiLineSample` asserts full Pos of each token; `TestLiterals`, `TestLongestMatchOperators` cover the classes | PASS |
| A test tokenizes a multi-line sample and asserts kind and position of each token | `TestTokenizeMultiLineSample` — a 5-line sample (incl. a blank line and a leading-tab line) with hand-computed Offset/Line/Col per token, element-by-element | PASS |
| Keywords distinguished from idents; contextual keywords stay IDENT | `TestKeywordVsIdent` (func/for/return/match/enum/assert → keyword kinds; implements/sealed/from/derive/foo → IDENT) | PASS |
| Stream ends with EOF | asserted in `TestTokenizeMultiLineSample` (last token EOF) and every table test (len==2: lexeme + EOF) | PASS |
| Unrecognized input → ILLEGAL, no panic | `TestIllegal` (`#` → ILLEGAL with Lit and Pos, lexing continues) | PASS |
| build / vet / test green | full suite run, all green | PASS |

No acceptance criterion lacks a covering test.

## Assumptions
- Column is a 1-based byte column; the multi-line test's expected values encode
  that (tab counts as one column).
- Comments are emitted as COMMENT tokens (TestComments), not dropped, so US-013
  can reclassify `///` and attach trivia.
