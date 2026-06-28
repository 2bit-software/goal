# Audit: Completeness — US-026

Scope is a narrow CLI wiring story over existing, unit-tested seams
(parser/sema/interp). Findings:

- MINOR: The spec does not say whether `--engine` accepts other values (e.g.
  `splice`). Resolved by FR-4: only `interp` selects the interpreter; anything
  other than `interp`/`ast` is rejected (covered by an acceptance criterion).
- MINOR: Output trailing newline semantics unspecified; the test trims
  whitespace, matching the existing `TestRunExecutesMain` convention.

No CRITICAL or MAJOR findings. Happy path (FR-1/2/3), opt-in default (FR-4), and
error cases (FR-5: parse/no-main/gate/runtime) are all covered with verifiable
acceptance criteria.

## Assumptions
- The interpreter path is FILE-based (one `.goal` file), consistent with
  `interp.New` operating on a single `*ast.File`. Directory/package runs stay on
  the transpile path.
- The CLI run uses full authority (`cap.GrantAll`, the `interp.New` default).
- `--engine=ast` (or absence) selects the existing transpile-and-`go run` path.
