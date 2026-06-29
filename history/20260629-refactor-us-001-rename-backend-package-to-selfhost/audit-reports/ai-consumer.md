# Audit: AI-Consumer Readiness — US-001

Could an AI implement this without guessing? Yes — the port pattern is documented
in progress.txt's Codebase Patterns block and demonstrated by 7 prior ports in
internal/selfhost/port_test.go.

## Findings

- All terms defined: BuildTranspiled (compile gate), BuildAndTest (behavioral
  gate), layout/deps keyed by module-relative dir, discoverPorted helper.
- Data formats specified: `map[string]*project.Package` keyed by "internal/<pkg>";
  testFiles are repo-relative paths copied verbatim.
- Acceptance criteria are test-assertable: a new TestPortedBackendPackage runs
  both gates; task fixpoint auto-covers the new package.
- No open questions block implementation.

## Assumptions surfaced

- The behavioral-gate test file list is exactly the 12 fixture-free tests; the
  corpus/fixture-dependent tests stay in internal/backend and run under task check
  (where the real fixtures exist), not in the throwaway temp module.
