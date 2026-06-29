# Plan Buildability Audit — US-007

## Findings

- Dependency order valid: token (already ported) -> ast. No forward refs.
- Interface contracts unchanged (verbatim copy of package ast).
- File paths verified against the real tree (selfhost/{token,lexer} exist;
  selfhost/ast is new; internal/ast/ast_test.go exists for the behavioral gate).
- Integration points are concrete: project.Discover + the existing
  selfhost.BuildTranspiled/BuildAndTest harness with the deps param from US-006.

No CRITICAL or MAJOR findings.

## Assumptions

- BuildAndTest's deps mechanism (added US-006) transpiles the token dep into the
  temp module so ast's `goal/internal/token` import resolves — confirmed by the
  lexer port using the identical shape.
