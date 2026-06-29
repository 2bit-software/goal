# Implementation Plan — US-008 Port parser package to goal

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| selfhost/parser/parser.goal | Ported copy of internal/parser/parser.go (recursive-descent core) |
| selfhost/parser/goal_construct.goal | Ported copy of internal/parser/goal_construct.go |
| selfhost/parser/goal_decl.goal | Ported copy of internal/parser/goal_decl.go (enum/sealed decls) |
| selfhost/parser/goal_match.goal | Ported copy of internal/parser/goal_match.go (match parsing) |
| selfhost/parser/goal_stmt.goal | Ported copy of internal/parser/goal_stmt.go (assert/from/derive/doctest) |

### Modified Files
| File | Change |
|------|--------|
| internal/selfhost/port_test.go | Add TestPortedParserPackage (compile + behavioral gates) |
| prd.json | Set US-008 passes: true (loop-runner step, post-verify) |
| progress.txt | Append US-008 entry (loop-runner step, post-verify) |

## Reuse
- selfhost.BuildTranspiled(layout) — compile gate over {token,lexer,ast,parser}.
- selfhost.BuildAndTest(relDir, pkg, testFiles, deps) — behavioral gate.
- project.Discover("../../selfhost/<pkg>") — load each ported package.
- Follows the exact shape of TestPortedAstPackage (the most recent prior port).

## Steps
1. Copy the 5 non-test parser source files verbatim to selfhost/parser/*.goal.
   (Go superset = valid goal; verified zero bare match/enum/assert identifier
   collisions — all occurrences are comments or token.MATCH/token.ENUM.)
2. Add TestPortedParserPackage:
   - Discover selfhost/{token,lexer,ast,parser}; assert package names.
   - BuildTranspiled over layout {internal/token, internal/lexer, internal/ast,
     internal/parser}.
   - BuildAndTest("internal/parser", parserPkg, ["../parser/parser_test.go"],
     deps={internal/token, internal/lexer, internal/ast}).
3. Run `go test ./internal/selfhost`, then `task check` and `task build`.

## Test Strategy
- Compile gate proves the transpiled parser + deps build as a Go module.
- Behavioral gate runs the self-contained parser_test.go against the transpiled
  source, proving behavioral equivalence (parse correctness, precedence,
  error handling).

## Risks
- Generated Go for the large parser.go may surface a transpile defect the checker
  is silent on — that is exactly what the compile gate catches. Mitigation: if it
  fails, inspect the generated Go and fix at the front-end/backend, but that would
  expand scope beyond this story and signal a BLOCKED return.
