# Plan Buildability Audit — US-006

## Checks
- Dependency order valid: source copy -> harness extension -> test. No forward refs.
- Interface contract concrete: `BuildAndTest(relDir, pkg, testFiles, deps)` with
  deps `map[string]*project.Package`; existing token call passes nil.
- File paths verified: `internal/selfhost/selfhost.go`, `internal/selfhost/port_test.go`,
  `selfhost/token/token.goal` all exist; `selfhost/lexer/` is new.
- Integration points specific: `project.Discover`, `BuildTranspiled`, `BuildAndTest`,
  `writePackage` are existing functions in internal/selfhost.
- `task fixpoint` auto-discovers selfhost/lexer (project.Discover walks the tree).

## Findings
No CRITICAL or MAJOR findings. Plan is executable as written.

## Assumptions
- `unicode`/`unicode/utf8` pass through as foreign imports unchanged (same class
  the harness already handles for other stdlib imports).
