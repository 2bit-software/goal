# Implementation Plan — US-002 sema package-check driver

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/sema/package.go` | The package driver: `AnalyzePackageInDir` + `AnalyzePackageInDirWith`. Parses each source, ResolvePackage, aggregates imports, EnrichForeign once, Check per file in input order. |
| `internal/sema/package_test.go` | Multi-file fixture test (cross-file exhaustiveness + foreign-enrichment-dependent unsourced-field) plus input-order and parse-error tests. |

### Modified Files
None. (Existing sema entry points are reused as-is.)

## Package Structure

```
internal/sema/
  package.go        (new)  — AnalyzePackageInDir / AnalyzePackageInDirWith
  package_test.go   (new)
  resolve.go               — ResolvePackage (reused)
  foreign.go               — EnrichForeign, DirResolver, DefaultResolver (reused)
  check.go                 — Check, Diagnostic (reused)
```

## Dependency Graph

1. `internal/parser` (ParseFile), `internal/ast` (File, ImportSpec) — exist.
2. `sema.ResolvePackage`, `sema.EnrichForeign`, `sema.Check` — exist (US-001).
3. `internal/sema/package.go` — depends on 1 + 2.
4. `internal/sema/package_test.go` — depends on 3 + the shared
   `internal/analyze/testdata/extpkg` fixture.

## Interface Contracts

```go
// AnalyzePackageInDir runs the AST checker over a multi-file goal package,
// returning one []Diagnostic per input file in input order. Foreign-import
// errors are non-fatal and discarded; a parse error is returned.
func AnalyzePackageInDir(srcs []string, dir string) ([][]Diagnostic, error)

// AnalyzePackageInDirWith injects the import resolver and surfaces the
// non-fatal per-import enrichment errors. A nil resolve uses DefaultResolver.
func AnalyzePackageInDirWith(srcs []string, dir string, resolve DirResolver) ([][]Diagnostic, []error, error)
```

Behavior of `...With`:
1. Parse each `src` -> `*ast.File`; return the parse error immediately on failure.
2. `info := ResolvePackage(files)`.
3. Aggregate `file.Imports` across all files into one `[]*ast.ImportSpec`.
4. `ferrs := EnrichForeign(info, imports, dir, resolve)`.
5. For each file in order, `out[i] = Check(files[i], info)`.
6. Return `out, ferrs, nil`.

## Integration Points

- Consumes `parser.ParseFile` (internal/parser) — new sema->parser edge; verified
  acyclic (parser deps = token/ast/lexer only).
- Reuses `ResolvePackage` (resolve.go:27), `EnrichForeign` (foreign.go:56),
  `Check` (check.go:58), `DirResolver` (foreign.go:43), `DefaultResolver`.
- Future consumer: cmd/goal checkPackage (US-004) — out of scope here.

## Testing Strategy

`internal/sema/package_test.go` (package `sema`, stdlib testing, no testify):

- `TestAnalyzePackageInDirCrossFileExhaustiveness`: enum in file A, non-exhaustive
  match in file B -> file B carries `non-exhaustive-match` Error, file A clean;
  proves cross-file merge + input order + per-file results (FR-1/FR-2).
- `TestAnalyzePackageInDirForeignEnrichedDeriveFinding`: file A declares local
  struct `Target{ID string; Extra string}`; file B declares
  `derive func make(o *ext.Outer) Target` importing the extpkg fixture via a fake
  resolver. With enrichment, ext.Outer resolves and `Target.Extra` is unsourced ->
  Error `unsourced-field` in file B (FR-3). A control assertion (no resolver / nil
  enrichment) shows the same derive defers to a Warning, proving the finding
  depends on foreign enrichment.
- `TestAnalyzePackageInDirParseErrorReturned`: a malformed source returns a
  non-nil error.

Resolver injection: a closure mapping `example.com/ext` ->
`filepath.Abs(../analyze/testdata/extpkg)`, mirroring foreign_test.go.
