# Plan Audit: Coverage — US-012

## Requirement → Plan trace

| Spec item | Plan element | Status |
|-----------|--------------|--------|
| FR-1 tokenize Go subset | lexer.go lexeme coverage (idents/keywords/numbers/strings/runes/operators/delimiters) | covered |
| FR-2 accurate positions | Offset/Line/Col semantics + running line/lineStart | covered |
| FR-3 keyword vs ident | `token.Lookup` → keyword Kind else IDENT; contextual keywords stay IDENT | covered |
| FR-4 whitespace + EOF + comments | whitespace skip, trailing EOF, COMMENT tokens | covered |
| Error handling (ILLEGAL) | unknown byte → ILLEGAL token | covered |
| AC: multi-line position test | `TestTokenizeMultiLineSample` | covered |
| AC: keyword/ident, contextual kept IDENT | `TestKeywordVsIdent` | covered |
| AC: EOF terminates | asserted in multi-line + EOF tests | covered |
| AC: build/vet/test green | verify gate | covered |

No requirement is unmapped. No plan element lacks a backing requirement —
comment recognition traces to FR-4; longest-match operators trace to FR-1.

## Findings

No CRITICAL or MAJOR findings. One MINOR:

### MINOR — Number-literal classification breadth
The plan describes one number-scan path classifying INT/FLOAT/IMAG. The corpus
is plain Go numbers, so full Go-spec underscore-digit grouping (`1_000`) is
nice-to-have, not required. Acceptable to implement the common forms and treat
exotic forms as a follow-up if a corpus case ever needs them.

## Assumptions

- Comment tokens are in scope as recognition only (not trivia attachment).
- The acceptance test's "sample" is author-chosen; it must be multi-line and
  cover each lexeme class to satisfy the AC meaningfully.
