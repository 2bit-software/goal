# Plan Coverage Audit — US-005

## Coverage check

- AC "selfhost/token holds the token package as goal source" -> new file
  `selfhost/token/token.goal`. Covered.
- AC "transpiles via US-002 smoke gate and generated Go compiles" -> port_test
  calls `selfhost.BuildTranspiled`. Covered.
- AC "existing token tests pass against the transpiled package" -> new
  `BuildAndTest` helper + port_test copying `internal/token/token_test.go`.
  Covered.
- AC "task check / task build stay green" -> project gates. Covered.

No scope creep: BuildAndTest is the minimal reusable harness needed; it is
justified as the reused verification path for US-006+.

No CRITICAL or MAJOR findings.

## Assumptions

- The verification test belongs in internal/selfhost alongside the existing
  smoke gate.
