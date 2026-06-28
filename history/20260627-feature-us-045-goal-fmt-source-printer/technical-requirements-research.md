# Technical Requirements / Research — goal fmt

## Hard constraint from prior stories (carry-forward)

The parser DROPS `//` and `/*` comments — they are not on the AST. Therefore an
AST-driven source rewriter/printer (goal fmt) MUST keep a raw-source comment guard
and splice by node Pos/End offsets, **never print the bare AST** (that would drop
every comment). This is recorded in progress.txt's Codebase Patterns and is the
governing design rule for this story.

## Chosen design

A position-driven reindenter, not an AST pretty-printer:

1. Parse the source with `internal/parser.ParseFile` purely as a validity gate —
   only well-formed goal is formatted (mirrors `internal/fix`'s conservative
   contract). Return the parse error otherwise.
2. Lex the source with `internal/lexer.Tokens` to get every token (including
   `COMMENT`/`DOC_COMMENT`) with line/offset positions. Compute leading-indent
   depth from delimiter nesting (`(`/`[`/`{` vs `)`/`]`/`}`) over the TOKEN stream
   — so braces inside comments or string literals never miscount.
3. Emit output line-by-line from the RAW SOURCE lines (not reconstructed from the
   AST/tokens): normalize only leading indentation (one tab per nesting depth,
   dedenting a line whose first token is a closer), strip trailing whitespace,
   collapse runs of blank lines to one, and end with exactly one newline.
   Multi-line string/comment continuation lines are kept verbatim.

This preserves comments, string contents, operator spacing, and struct-field
alignment verbatim (only leading/trailing whitespace and blank runs are touched),
which makes it both comment-preserving and idempotent by construction: re-lexing
the output yields the same token lines and the same depths, so a second pass is a
no-op.

## Packages / files

- New `internal/goalfmt` package: `Source(src string) (string, error)`.
- `internal/goalfmt/format_test.go`: idempotency over every corpus `.goal` input
  (via `internal/corpus` Load + CaseInputs) plus a comment-retention unit test.
- `cmd/goal`: add the `fmt` subcommand (`goal fmt [-w] [path]`), registered in
  `guideCommands`, dispatched in `run`.

## No specific technology beyond the existing stdlib-only constraint.
