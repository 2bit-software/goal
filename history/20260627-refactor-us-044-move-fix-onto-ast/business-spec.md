# Business Spec — US-044 Move goal fix onto the AST

## Outcome

`goal fix` (the idiomatize rewriter) continues to behave exactly as today from
the user's perspective: the same source-to-source rewrites are produced, the
same opportunities are reported, and untouched code (formatting, comments) is
preserved byte-for-byte.

## Requirements

- internal/fix detects rewrite candidates from the parsed AST, not from a
  re-lexed token stream.
- The four rules are preserved with identical observable behavior:
  - convert a plain Go `(T, error)` function into a `Result[T, error]` function
    (signature + `return v, nil` → `return Result.Ok(v)`);
  - collapse manual error/nil propagation inside a Result/Option function into
    the `?` operator (both the value-binding form and the `if err := f(); ...`
    init-guard form), rewriting later `*o` dereferences;
  - report a `switch` over an in-file enum as a `match` candidate (no rewrite);
  - report manual error handling inside a non-Result function as a call-site.
- Conservative refusals are preserved: a decorated/wrapped error, a non-zero
  return, an `else`, a comment inside a collapsible block, and a multi-value
  tuple all leave the code untouched and record the appropriate report.
- `fix.File` stays idempotent.

## Constraints

- Zero new dependencies; stdlib testing only.
- Public API of internal/fix (File, Change, Report, Level and its constants)
  is unchanged — cmd/goal and internal/lsp consume it as-is.
