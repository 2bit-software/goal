# Plan Audit

## Findings

No CRITICAL or MAJOR findings. The plan is implementable as written.

- MINOR: The plan should ensure the formatter's output for a directory walk skips
  non-`.goal` files and the `attic/` tree. Mitigated by mirroring `cmdFix`'s
  existing discovery (which already filters to `*.goal`).
- MINOR: Idempotency depends on the lexer not emitting tokens whose `Pos.Line`
  disagrees with `strings.Split` line indexing. Verified consistent (lexer line is
  1-based, incremented on `\n`; Split index = line-1). The corpus idempotency test
  will catch any discrepancy.

## Traceability check

Every FR/AC traces to a plan element (see plan.md "Requirement traceability"). No
circular dependencies: goalfmt → parser/lexer/token only; the test → corpus (one
direction). File paths verified to exist (`cmd/goal/main.go`,
`internal/{parser,lexer,token,corpus}`).

## Assumptions

- The corpus manifest path/repo-root depth from `internal/goalfmt` is `../..` (same
  as `internal/corpus`), consistent with existing tests.
- Regenerating `AI-KNOWLEDGE-BOOTSTRAP.md` after adding the `fmt` subcommand is
  required to keep `TestBootstrapGoldenMatches` green.
