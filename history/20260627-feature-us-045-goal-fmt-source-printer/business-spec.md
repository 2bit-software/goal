# Business Spec — goal fmt source printer (US-045)

## Outcome

goal developers get a canonical, deterministic source format for `.goal` files,
the way `gofmt` gives Go a canonical format. A `goal fmt` command formats source
predictably so diffs stay small and style debates disappear.

## Functional Requirements

- FR-1: The system SHALL provide a `goal fmt [path]` command that formats a single
  `.goal` file or every `.goal` file under a directory and prints the formatted
  source to stdout.
- FR-2: The system SHALL provide a `goal fmt -w [path]` mode that writes the
  formatted result back to each file in place instead of printing it.
- FR-3: Formatting SHALL preserve every comment present in the input (`//` line
  comments and `///` doc comments).
- FR-4: Formatting SHALL be idempotent: formatting already-formatted source SHALL
  produce byte-identical output. For all source `s`, `fmt(fmt(s)) == fmt(s)`.
- FR-5: Formatting SHALL preserve program meaning — source that parses before
  formatting SHALL parse to the identical program after formatting.
- FR-6: Formatting SHALL only normalize layout: leading indentation, trailing
  whitespace, and runs of blank lines. It SHALL NOT reflow or rewrite token text,
  string contents, or intra-line spacing.

## Acceptance Criteria

- AC-1 (happy path): For every `.goal` input in the corpus manifest,
  `fmt(fmt(src)) == fmt(src)` (idempotency holds across the whole corpus).
- AC-2 (comments): For a representative sample containing `//` and `///` comments,
  every comment from the input is present in the formatted output.
- AC-3 (error path): Source that does not parse SHALL cause `goal fmt` to report an
  error and leave the file untouched (no partial or guessed formatting).
- AC-4 (meaning preserved): Formatted corpus output SHALL still parse without error.

## Error Handling

- Unparseable `.goal` source: report the parse error to stderr; do not modify the
  file. The exit status is non-zero for that file.
- I/O errors (bad path, unreadable/unwritable file): reported as an operational
  failure; the command exits non-zero.

## Out of Scope

- Column alignment of struct fields / `=>` arms beyond what the input already has
  (the formatter preserves existing intra-line alignment verbatim but does not
  compute new alignment).
- Reflowing long lines, sorting imports, or any AST-level rewrite (that is `goal
  fix`'s domain, not `goal fmt`).
- Switch-case body dedent styling (the corpus has no `switch`; indentation is by
  pure delimiter nesting).

## Open Questions

- None blocking. Default behavior (print to stdout, `-w` to write) mirrors the
  existing `goal fix` command's flag shape, so no user input is required.
