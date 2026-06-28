# Scaffold notes — US-009

## Changed

- `internal/backend/package.go`:
  - Dropped the `goal/internal/analyze` import.
  - `enrichForeign(info, srcs, dir)` -> `enrichForeign(info, files []*ast.File, dir)`.
    Now aggregates every file's `ast.File.Imports` and calls
    `sema.EnrichForeign(info, imports, dir, nil)` directly.
  - Removed the redundant `srcs []string` slice in `TranspilePackage` (only fed
    the old analyze-based helper); foreign loading is now AST-import driven.
  - Updated the package-doc comment to drop analyze references.

## Why this is behavior-preserving (or strictly better)

- In-package facts (Structs, FromRegistry, FuncSignatures, enums, sealed) already
  came from `sema.ResolvePackage(files)`; the old analyze copy only re-added
  Structs (keyed bare, skipped when in-package) and FromRegistry (already present
  from sema). Both were redundant for in-package and analyze.EnrichForeign never
  added foreign FromRegistry entries.
- Foreign structs from sema.EnrichForeign are keyed `alias.Type` — no collision
  with bare in-package names — matching what analyze produced. sema.EnrichForeign
  additionally populates FuncSignatures + ForeignMethods (more facts, not fewer).

## Test independently

- `task check` (go vet + full `go test ./...`) and `task build`.
- Conformance: `go test ./internal/corpus/...` (behavioral package-mode tier).
- Import check: `grep -rn "internal/analyze" internal/backend` returns nothing.
