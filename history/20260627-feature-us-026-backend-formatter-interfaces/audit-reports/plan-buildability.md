# Plan Audit: Buildability — US-026

## Dependency order — valid

1. `internal/sema` (ast only) → 2. `internal/backend` (ast, parser, sema,
   pipeline) → 3. `cmd/goal` wiring → 4. tests. No forward references.

## Import-cycle check — clear

Verified against the codebase: `internal/corpus` imports check, parser,
pipeline, project — NOT `internal/backend`. Nothing in the repo imports
`internal/backend`. Therefore:
- `internal/backend` may import `internal/pipeline` (pipeline doesn't import
  backend).
- `internal/backend`'s external test (`package backend_test`) may import
  `internal/corpus` (corpus doesn't import backend).
No cycles.

## Interface contracts — concrete

Signatures for `Backend.Emit`, `Formatter.Format`, `GoFormatter`, `goBackend`,
and `Transpile` are given in actual Go syntax and agree across components
(`Transpile` returns `pipeline.Output`, matching `corpus.Transpiler`).

## File paths — verified

`internal/{sema,backend}` do not exist yet (no conflict). `cmd/goal/main.go`,
`internal/corpus/behavior_runner.go`, `internal/parser`, `internal/ast`,
`internal/pipeline` all exist as referenced.

## Findings

- MINOR: The minimal emitter's exact node coverage is left to the implementer.
  Risk is contained: the fixture is implementer-chosen, so coverage and fixture
  are co-designed. Unsupported nodes return a descriptive error (correct for a
  seam story).

No CRITICAL/MAJOR findings. The plan is buildable in order.

## Assumptions

- `parser.ParseFile` returns a usable `*ast.File` for the chosen plain-Go fixture
  (true since US-024 parses 100% of corpus).
- A minimal recursive Go emitter covering package/import/FuncDecl/GenDecl + basic
  stmts/exprs/types suffices for the fixture and gofmt-parses.
