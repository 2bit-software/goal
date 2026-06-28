# Research — goal fmt source printer

This is an internal-codebase task; no external/web research is required. Findings
come from reading the existing front-end.

## Key findings from the codebase

- `internal/parser.ParseFile(src) (*ast.File, error)` is the AST gate. The parser
  RETAINS only `DOC_COMMENT` in the stream and DROPS `//` (`COMMENT`) and `/*`
  comments — they are not represented on the AST. (progress.txt Codebase Patterns;
  `internal/fix/fix.go` `spanHasComment` exists precisely because comments are
  invisible in the AST.)
- `internal/lexer.Tokens(src) []token.Token` returns every token INCLUDING
  `COMMENT` and `DOC_COMMENT`, each carrying `token.Pos{Offset, Line, Col}`. A
  comment is a single token, so delimiters inside comments/strings are not
  separate tokens — counting delimiter KINDS over the token stream ignores braces
  inside comments/strings automatically (raw char-counting would not).
- `internal/fix` is the established precedent for AST-aware, comment-preserving
  source rewriting: it parses for structure but emits minimal byte splices and
  never prints the AST (which would reflow/drop comments).

## Corpus survey (idempotency surface)

- 108 `.goal` inputs; 69 contain `//`/`///` comments.
- NO `/* */` block comments, NO raw (backtick) string literals (every backtick in
  the corpus is inside a comment), NO `switch` statements. So multi-line-token
  protection and switch-case dedent are not exercised by the corpus, though the
  formatter still guards multi-line tokens defensively.

## Decision

Token-driven reindenter (see technical-requirements-research.md). Output is built
from raw source lines with only leading-indent / trailing-whitespace / blank-run
normalization, which is comment-preserving and idempotent by construction.

**Confidence: High** — the design touches only insignificant whitespace, so the
token stream (hence the AST and program meaning) is invariant across formatting,
and a second pass re-derives identical indentation.
