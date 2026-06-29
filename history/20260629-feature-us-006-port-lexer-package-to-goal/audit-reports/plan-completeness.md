# Plan Coverage Audit — US-006

## Mapping

| Acceptance criterion | Plan element |
|----------------------|--------------|
| selfhost/lexer holds the lexer as goal source importing ported token | New file `selfhost/lexer/lexer.goal` (copy of internal/lexer/lexer.go) |
| It transpiles and generated Go compiles (unicode/utf8 pass through) | Compile gate: `BuildTranspiled` over {token, lexer} |
| Existing lexer tests pass against transpiled package | Behavioral gate: `BuildAndTest("internal/lexer", lexerPkg, ["../lexer/lexer_test.go"], {token})` |

No scope creep: the only non-test change beyond the source copy is the
BuildAndTest deps extension, which is required for criterion 3.

## Findings
No CRITICAL or MAJOR findings.

## Assumptions
- The token package (US-005) is the sole in-module dependency of lexer.
