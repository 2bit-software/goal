# Technical Requirements / Research — US-009

## Established port pattern (US-005..US-008)

Port = copy internal/<pkg>/*.go (non-test) -> selfhost/<pkg>/*.goal verbatim
(Go superset is valid goal), then add a port_test in internal/selfhost that:
- COMPILE gate: selfhost.BuildTranspiled(layout) over the package + all its
  in-module deps, keyed by module-relative dir ("internal/<pkg>").
- BEHAVIORAL gate: selfhost.BuildAndTest("internal/sema", pkg, testFiles, deps)
  transpiles the package + deps into a temp `module goal`, copies the chosen
  EXISTING white-box test files beside the generated Go, runs `go test`.

## sema specifics

- internal/sema depends on token, ast, parser (in-module) and pass-through
  foreign imports go/parser, go/format, go/types (foreign.go).
- Reserved-word scan (bare match/enum/assert identifiers): none — all hits are
  string literals, so files port verbatim.
- Non-test sources to copy: analyze, assert, check, convert, fields, foreign,
  implements, mustuse, package, question, resolve, sema (.go -> .goal).
- Behavioral gate must use only self-contained test files (no repo-relative
  ../../features fixtures, no testdata/extpkg dir, no dropped symbols), mirroring
  US-007/US-008 which excluded fixture-dependent suites.

## Verification

- task check, task build (prd verifyCommands)
- go test ./internal/selfhost (the new sema gate)
- task fixpoint (selfhost/sema auto-covered by project.Discover)
