# Research — US-002

This story composes existing internal seams; no external/library research needed.

## Findings (from the codebase)

- Legacy reference shape: `internal/check/check.go` `AnalyzePackageInDir` /
  `AnalyzePackageInDirWith` (lines 169-198). It builds analyze.Tables once,
  enriches foreign, then runs per-file `Run` returning `[][]Diagnostic` in input
  order. The sema driver mirrors this exactly so US-004 is a near drop-in swap.

- sema entry points already exist:
  - `parser.ParseFile` (internal/parser) — parse a source string to *ast.File.
  - `sema.ResolvePackage([]*ast.File) *Info` (resolve.go:27) — merged per-file facts.
  - `sema.EnrichForeign(info, imports, dir, resolve) []error` (foreign.go:56) —
    US-001 foreign enrichment, driven by `ast.File.Imports`, non-fatal errors.
  - `sema.Check(file, info) []Diagnostic` (check.go:58) — all AST checks per file.
  - `sema.DirResolver` / `sema.DefaultResolver` (foreign.go) already exported.

- Import-cycle safety: `go list -deps ./internal/parser` = token, ast, lexer
  only — parser does NOT import sema, so sema may import parser. (corpus and
  backend already do `parser.ParseFile` + `sema.*`.)

- AC-3 foreign-dependent finding: reuse internal/analyze/testdata/extpkg
  (ext.Outer{ID string, Count int, ...}). A `derive func make(o *ext.Outer)
  Target` whose Target (sibling file) carries a field absent from ext.Outer
  yields the Error `unsourced-field` only once EnrichForeign loads ext.Outer's
  fields (otherwise CheckConvert defers with a Warning). This single finding
  exercises both cross-file resolve and foreign enrichment.

## Confidence: High — all seams verified in-tree.
