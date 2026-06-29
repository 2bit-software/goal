# Tasks — US-005 Port token package to goal

## T1: Add selfhost/token goal source
- **Files**: `selfhost/token/token.goal` (new)
- **Action**: Copy `internal/token/token.go` verbatim as goal source (package
  token). No semantic edits; reserved-word safety already confirmed.
- **Spec coverage**: FR-1, AC-1.
- **Verify (DONE)**: `task build` (front-end parses it during discovery) stays green;
  file present and declares `package token`.

## T2: Add reusable BuildAndTest harness helper
- **Files**: `internal/selfhost/selfhost.go` (modify)
- **Action**: Add `BuildAndTest(relDir string, pkg *project.Package, testFiles
  []string) error` — transpile pkg into a temp `module goal` at relDir, copy
  each testFile into that dir, run `go test ./<relDir>`.
- **Spec coverage**: FR-3 (enabler).
- **Verify (DONE)**: compiles (`task check`).

## T3: Add ported-token verification test
- **Files**: `internal/selfhost/port_test.go` (new)
- **Action**: Discover `selfhost/token`, assert one `token` package, run
  `BuildTranspiled` (compiles) and `BuildAndTest` with
  `../token/token_test.go` (existing tests pass).
- **Spec coverage**: FR-2, FR-3, AC-2, AC-3.
- **Verify (DONE)**: `go test ./internal/selfhost` passes; whole suite via `task check`.

## T4: Project gates
- **Action**: Run `task check` and `task build`; confirm bootstrap/fixpoint
  unaffected (selfhost/token auto-discovered, emitted identically).
- **Spec coverage**: AC-4.
- **Verify (DONE)**: both gates green.
