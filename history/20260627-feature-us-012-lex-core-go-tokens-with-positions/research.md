# Research — US-012 Lexer for the Go subset

## Question

How should `internal/lexer` turn goal source into `[]token.Token` with accurate
`Pos{Offset, Line, Col}` for the Go subset (idents, keywords, literals,
operators, delimiters)?

## Canonical reference: Go's own scanner

- **What it is**: `go/scanner` + `go/token` in the Go standard library — the
  authoritative hand-written lexer for the exact grammar we target a subset of.
- **Design**: a struct cursor over `[]byte` source. Fields: `src`, `offset`
  (start of current rune), `rdOffset` (read position), `ch` (current rune),
  plus a line/column tracker. `next()` advances one rune; `scanIdentifier`,
  `scanNumber`, `scanString`, `scanRune`, `scanComment` consume each lexeme.
  Operators use a `switch ch` with longest-match nested switches
  (`<` → `<=` / `<<` / `<<=` / `<-`).
- **Positions**: Go tracks line offsets in a `token.File`; columns derive from
  `offset - lineStart + 1`. Our `token.Pos` carries Offset/Line/Col directly, so
  the lexer keeps a running `line` and `lineStart` (byte offset of the current
  line's first byte) and computes `Col = offset - lineStart + 1`.
- **Evidence**: `src/go/scanner/scanner.go` in the Go tree; behavior matches the
  Go spec lexical elements.

## Decisions for this story

- Operate over `[]byte(src)`; cursor by byte offset (ASCII operators), but treat
  identifier/letter classification with `unicode.IsLetter`/`IsDigit` so non-ASCII
  idents tokenize (matches Go). `Col` is a byte column (consistent, simple, and
  what the corpus — pure ASCII — needs).
- Longest-match operators by reusing the multi-char `Kind`s already in
  `internal/token` (`SHL_ASSIGN`, `AND_NOT`, etc.).
- Recognize `//` and `/* */` comments and emit them as `COMMENT` tokens with
  their text in `Lit`, so US-013 can reclassify `///` and attach trivia. (For
  US-012 the test sample is comment-free; comments are recognized, not dropped.)
- Emit a trailing `EOF` token at the end-of-input position.
- Keep `?`, `=>`, `...` OUT — deferred to US-013. `?` and a bare `=>`/`...` in
  source would lex as ILLEGAL / separate tokens here; the US-012 sample uses
  none.

## Confidence

High — this is a direct, smaller port of the well-understood `go/scanner`
design onto the existing `internal/token` vocabulary.

## Open questions

None blocking. Comment-trivia attachment and goal lexemes are explicitly US-013.
