# Plan Audit — Buildability (US-002)

- Dependency order is valid: parser/ast (exist) -> sema seams (exist) ->
  package.go -> package_test.go. No forward references.
- Interface contracts use real signatures; types match the reused functions
  (ResolvePackage([]*ast.File)*Info, EnrichForeign(info,imports,dir,resolve)[]error,
  Check(file,info)[]Diagnostic).
- File paths verified: internal/sema/{package.go,package_test.go} do not exist
  yet; the extpkg fixture exists at internal/analyze/testdata/extpkg.
- Import cycle: sema->parser is acyclic (parser deps = token/ast/lexer).
- Each component compiles in order; the test compiles only after package.go.

No CRITICAL/MAJOR findings.

## Assumptions
- Imports aggregated across all files before a single EnrichForeign call; the
  merged Info is shared by every per-file Check.
- The foreign-dependent control assertion uses a nil/empty enrichment path to
  contrast the Warning (deferred) vs Error (enriched) outcome.
