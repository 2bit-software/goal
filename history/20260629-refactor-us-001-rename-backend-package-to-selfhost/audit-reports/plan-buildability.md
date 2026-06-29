# Plan Audit: Buildability — US-001

- Dependency order valid: backend's deps (token, lexer, ast, parser, sema,
  project, pipeline) are all already ported under selfhost/ — no forward refs.
- Interface contracts: TestPortedBackendPackage reuses the exact signatures of
  BuildTranspiled(map[string]*project.Package) and BuildAndTest(relDir, pkg,
  testFiles, deps) verified in selfhost.go — types match.
- File paths verified: selfhost/backend/ is new; the 6 source files exist in
  internal/backend; port_test.go exists and has discoverPorted.
- Each step compiles incrementally; a PostToolUse `task check` hook may report
  transient failures mid-refactor (expected) — only the final green state matters.

Open risk: splitting backend_test.go could leave an unused import in the original
file. Mitigation: run task check after the split and remove any now-unused import.
No CRITICAL/MAJOR findings.
