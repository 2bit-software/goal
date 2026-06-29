# Research Findings — US-002

## How package transpile + build already works

- `backend.TranspilePackage(*project.Package) (pipeline.PackageOutput, error)` is
  the package driver: parses each `project.File`, `sema.ResolvePackage`, emits one
  Go file per source file (`goName`: `foo.goal -> foo.go`) plus a shared
  `goal_prelude.go` / `goal_options.go` when needed.
- `internal/corpus/package_runner.go` is the canonical transpile-then-build proof:
  build a `project.Package`, transpile, assert each file is valid Go via
  `go/format`, then write the generated files into a throwaway temp module and run
  `go build ./...`. This is the pattern to reuse.

## Dependency closure (decisive for strategy)

`go list -deps` over the 8 covered packages shows their non-test internal
dependency closure is **entirely within the covered set**
{token, lexer, ast, parser, sema, project, pipeline, backend}. So a single
self-contained temp module that mirrors `internal/<pkg>/` for all 8 transpiled
packages builds with no extra wiring (stdlib imports pass through). This beats the
`-overlay`-the-real-tree approach: full isolation, nothing touches the source tree.

## Gate strategy chosen

A Go test (stdlib `testing`) that, for each covered package, reads its non-`_test.go`
`*.go` files as goal source, transpiles via `backend.TranspilePackage`, writes the
generated Go into a temp module under `internal/<pkg>/`, and runs `go build ./...`.
Build failure => test failure. Plus a negative test proving the gate fails when a
package transpiles to non-compiling Go.

## DEFECT DISCOVERED — PRD premise was wrong

The PRD note claims "the compiler source is already valid goal." It is **not**:
goal reserves `match`, `enum`, `assert` beyond Go's keywords (contextual keywords
implements/sealed/from/derive are NOT reserved). Two compiler files use `enum` as a
plain local identifier — valid Go, rejected by the goal parser:

- `internal/sema/check.go`: `enum := info.Enums[...]` in `checkOneMatch`, and the
  `missingVariants(enum *Enum, ...)` helper.
- `internal/backend/emit.go`: `enum, ok := x.Enum.(*ast.Ident)` in `variantLit`,
  and a second `enum, ok := b.Enum.(*ast.Ident)` in the `Enum.Variant` switch.

The other six packages transpile and `go build` clean today. Making the gate green
requires renaming these `enum` identifiers (behavior-preserving) — exactly the class
of silent defect this gate exists to catch. `enumOf`, `enumName`, `Enums` are fine
(only the exact word `enum` is the keyword; the lexer takes the longest identifier).

## Scope decision

Include the two `enum` renames as part of standing up a *green* gate (AC: "green
after US-001"). Keep the gate to `go build` (the defect detector); running each
package's existing tests against the transpiled output is "where practical" and is
the explicit job of the later per-package port stories (US-005+).
