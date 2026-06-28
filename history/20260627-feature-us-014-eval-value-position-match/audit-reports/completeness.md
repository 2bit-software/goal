# Audit — Completeness

## Findings

- **MINOR** — The spec does not state evaluation order of scrutinee vs arms. The
  natural and only sensible order (scrutinee once, then dispatch) matches US-013;
  no ambiguity in practice.
- **MINOR** — Nested value-position match (`=> match { ... }`) is not called out.
  It is covered for free because the arm body is evaluated through `evalExpr`,
  which dispatches `*ast.MatchExpr` recursively. No action needed.

No CRITICAL or MAJOR findings. The spec covers the happy path (three positions),
payload-bearing arms, the rest arm, the loud-default error case, and the
unchanged statement path.

## Assumptions

- Value-position arm bodies are expressions (`=> expr`); a block/statement body
  in value position is a descriptive refusal rather than an attempt to extract a
  value from a statement list. This matches the parser's arm dispatch.
- The defensive-default panic reuses the same `panicSignal`/"unreachable" message
  as statement-position match.
