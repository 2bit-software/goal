# Verification — US-009

All acceptance criteria pass:
- AC1: selfhost/sema holds 12 .goal sources importing token/ast/parser. PASS
- AC2: transpiles + generated Go compiles (go/parser/format/types pass through)
  via BuildTranspiled. PASS
- AC3: existing self-contained sema suites pass against the transpiled package
  via BuildAndTest. PASS
- Project gates: task check green; task build green; task fixpoint FIXPOINT OK
  (selfhost/sema byte-identical across goal-c-1/goal-c-2).

Commit: 52f8fd4.

No CRITICAL/MAJOR/MINOR findings.

## Assumptions
- Behavioral gate covers the self-contained suites; foreign_test.go and
  package_test.go are excluded (testdata/extpkg fixture absent from the temp
  module) — consistent with US-007/US-008 precedent.
- lexer is included in the layout/deps because the transpiled parser imports it,
  though sema does not import lexer directly.
