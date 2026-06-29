# Plan Buildability Audit — US-008

## Buildability check
- Dependency order valid: token/lexer/ast are already ported (US-005/006/007) and
  present under selfhost/; parser imports only those + stdlib (errors, fmt).
- Interface contracts agree: parser.go's ast.* symbol usage is fully covered by the
  ported ast package (verified; only the dropped ast.Sexpr is unused by parser and
  by parser_test.go).
- File paths verified: selfhost/parser/ is new, no conflict; internal/selfhost/
  port_test.go exists and the new test mirrors TestPortedAstPackage exactly.
- Integration points specific: harness funcs BuildTranspiled/BuildAndTest with exact
  layout/deps keys ("internal/token", "internal/lexer", "internal/ast",
  "internal/parser") and testFile path "../parser/parser_test.go".

## Findings
No CRITICAL. No MAJOR. No MINOR.

## Assumptions
- The temp `module goal` mechanics and go.mod are handled by the existing harness;
  no new harness code required.
