# Scaffold notes — US-007

This refactor is an in-place fact-source swap (no parallel module makes sense for
a single internal package's private facts), done as minimal targeted edits.

## Changed files

- internal/typecheck/typecheck.go
  - `Package.Tables *analyze.Tables` -> `Package.Sema *sema.Info` plus a private
    `goalFiles []*goalast.File` (parsed goal ASTs, aligned with Src.Files).
  - `Load` now parses each file with the goal `parser.ParseFile` and builds facts
    via `sema.ResolvePackage(goalFiles)` instead of `analyze.BuildPackage(srcs)`.
    (Same parse backend.TranspilePackage already did, so it cannot newly fail.)
  - Removed dead `goalPosition` / `offsetLineCol` (only the implements check used
    them; it now reads token.Pos Line/Col off the AST). Dropped `analyze` and
    `strings` imports; added aliased `goalast`/`goalparser`.

- internal/typecheck/mustuse.go
  - `p.Tables.FuncSignatures` -> `p.Sema.FuncSignatures`, `analyze.ModeResult` ->
    `sema.ModeResult`, `p.Tables.Structs` -> `p.Sema.Structs`. Dropped `analyze`
    import (sema already present).

- internal/typecheck/implements.go
  - `p.Tables.Sealed` -> `p.Sema.Sealed`.
  - Replaced `scan.Lex`-based `implementsClauses(src string)` + `splitList` with an
    AST walk `implementsClauses(*goalast.File)` over top-level type decls reading
    `StructType.Implements` (interface off `ImplementsClause.Type` via new
    `ifaceName`, position off `ImplementsClause.Implements`). Dropped `scan` and
    `textedit` imports; added aliased `goalast`/`goaltoken`.

- internal/typecheck/typecheck_test.go
  - `p.Tables.FuncSignatures["greet"]` -> `p.Sema.FuncSignatures["greet"]`.

## How to test

`task check` (go vet + full `go test ./...`) and `task build`. The depth-check
tests (implements_test.go, mustuse_test.go, nozero_test.go, typecheck_test.go,
checker_test.go) plus the corpus conformance tier cover the behavior; positions,
codes, severities, and messages are unchanged.

## Behavior note

goal source admits one interface per `implements` clause (parser models a single
`Type Expr`); the old token scanner's comma-list path was unreachable capability.
No live behavior is lost.
