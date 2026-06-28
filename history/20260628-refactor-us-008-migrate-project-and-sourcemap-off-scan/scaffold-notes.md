# Scaffold notes — US-008

In-place migration of private leaf helpers (no public API change), so old and
new could not meaningfully coexist as separate symbols — the functions were
rewritten in place and verified by the existing tests.

## internal/project/project.go
- `PackageClause` now parses with `parser.ParseFile` and returns `file.Name.Name`,
  guarded by confirming `src` at `file.Package.Offset` begins with `"package"`
  (the parser synthesizes a name token even when the clause is missing).
- Dropped imports `goal/internal/scan` and `goal/internal/textedit`; added
  `goal/internal/parser`.

## internal/pipeline/sourcemap.go
- `declSites` now parses with `parser.ParseFile` and walks `file.Decls`, using
  `decl.Pos().Offset` as the line-start keyword offset (confirmed the goal parser
  always sets `FuncType.Func`, so methods too anchor at the `func` keyword).
- `declName` rewritten to take an `ast.Decl`; removed the token-based `declName`
  and `isDeclKeyword`.
- Dropped imports `goal/internal/scan` and `goal/internal/textedit`; added
  `goal/internal/ast`, `goal/internal/parser`, `goal/internal/token`.

## Verification
- `task check` (go vet + full `go test ./...`): green.
- `task build`: green.
- `grep internal/scan|internal/analyze` over both packages: no matches.

## Independent testability
Covered by existing `internal/project/project_test.go` (incl.
`TestPackageClauseIgnoresStringsAndComments`) and
`internal/pipeline/sourcemap_test.go` (`TestAddLineDirectivesAnchorsUserDecls`).
