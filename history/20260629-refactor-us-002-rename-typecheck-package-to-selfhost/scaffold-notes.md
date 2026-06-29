# Scaffold Notes — US-002

## New files created

- selfhost/typecheck/checker.goal
- selfhost/typecheck/implements.goal
- selfhost/typecheck/mustuse.goal
- selfhost/typecheck/nozero.goal
- selfhost/typecheck/typecheck.goal

All five are byte-for-byte copies of the corresponding internal/typecheck/*.go
files (verified with `diff -q`). No source edits were needed: the only
match/enum/assert occurrences are inside string-literal error text
(mustuse.go:93,194), not identifiers.

## Harness change

- internal/selfhost/port_test.go: added TestPortedTypecheckPackage using the
  existing discoverPorted helper. COMPILE gate = BuildTranspiled over the 9-entry
  layout {token,lexer,ast,parser,sema,project,pipeline,backend,typecheck};
  BEHAVIORAL gate = BuildAndTest("internal/typecheck", ...) over the 8-entry dep
  closure with the five white-box typecheck test files.

## How old and new coexist

internal/typecheck is untouched and still builds/tests under the Go toolchain.
selfhost/typecheck is .goal-only (invisible to the Go toolchain), so no package
collision. selfhost/typecheck is auto-discovered by `task fixpoint` via
project.Discover — no fixpoint harness change needed.

## How to test independently

go test ./internal/selfhost -run TestPortedTypecheckPackage

Result: PASS (both compile and behavioral gates green).
