# Verify — Acceptance Criteria — US-008

## Criterion-by-criterion
- [x] selfhost/parser holds the parser as goal source importing the ported token,
      lexer, ast packages — 5 .goal files present; imports goal/internal/{ast,lexer,token}.
- [x] It transpiles and the generated Go compiles — TestPortedParserPackage's
      BuildTranspiled gate (token+lexer+ast+parser) passes.
- [x] The existing parser tests pass against the transpiled package — BuildAndTest
      runs parser_test.go (the self-contained behavioral suite) against the
      transpiled source; passes.
- [x] Project gates green — `task check` (full go test ./...) and `task build` both
      green; `task fixpoint` reports FIXPOINT OK.

## Result
All acceptance criteria satisfied. No CRITICAL or MAJOR findings.

## Assumptions
- Behavioral gate scoped to parser_test.go; the fixture-reading and Sexpr snapshot
  suites are out of scope for the isolated temp module (documented in spec).
