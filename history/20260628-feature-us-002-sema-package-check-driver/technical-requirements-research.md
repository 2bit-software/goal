# Technical Requirements / Research — US-002

## Existing seams to compose

- `parser.ParseFile(src string) (*ast.File, error)` — parse each source file.
- `sema.ResolvePackage(files []*ast.File) *Info` — merge per-file facts.
- `sema.EnrichForeign(info *Info, imports []*ast.ImportSpec, dir string, resolve DirResolver) []error`
  (US-001) — fold imported Go package struct/method facts in. Non-fatal errors.
- `sema.Check(file *ast.File, info *Info) []Diagnostic` — run all AST checks per file.

## Shape (mirror legacy internal/check/check.go)

- `AnalyzePackageInDir(srcs, dir) ([][]Diagnostic, error)` delegates to
  `AnalyzePackageInDirWith(srcs, dir, nil)` (nil resolver = DefaultResolver).
- `AnalyzePackageInDirWith(srcs, dir, resolve)` parses each file, ResolvePackage,
  aggregates `file.Imports` across all files, EnrichForeign once over the merged
  info, then `Check` per file in input order.

## Import-cycle check

parser depends only on token/ast/lexer — does NOT import sema. So sema may import
parser without a cycle (verified via `go list -deps ./internal/parser`).

## Foreign-enrichment-dependent finding (AC 3)

Reuse the shared fixture internal/analyze/testdata/extpkg (ext.Outer has fields
ID string, Count int, ...). A `derive func make(o *ext.Outer) Target` where the
local Target (declared in a SIBLING file) has a field absent from ext.Outer:
- Without enrichment: source ext.Outer unresolved -> Warning unresolved-derive-type.
- With enrichment: ext.Outer fields load -> unsourced target field -> Error
  unsourced-field. This finding depends on both cross-file resolve AND foreign
  enrichment.
