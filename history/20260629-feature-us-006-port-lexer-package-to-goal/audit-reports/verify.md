# Verification — US-006

## Acceptance criteria

1. selfhost/lexer holds the lexer as goal source importing the ported token —
   DONE: selfhost/lexer/lexer.goal, `import "goal/internal/token"`.
2. Transpiles and generated Go compiles (unicode/utf8 pass through) — PASS:
   `go test ./internal/selfhost -run TestPortedLexerPackage` BuildTranspiled gate
   green over {token, lexer}.
3. Existing lexer tests pass against the transpiled package — PASS: BuildAndTest
   ran ../lexer/lexer_test.go against the transpiled lexer (token transpiled in
   as dep), green.

## Project gates
- `task check` — green (go vet + full test suite, incl. internal/selfhost).
- `task build` — green (bin/goal, bin/goalc).
- `task fixpoint` — FIXPOINT OK; goal-c-1 and goal-c-2 emit
  selfhost/lexer/lexer.go byte-identically.

## Findings
None. No CRITICAL or MAJOR. Feature works as specified.
