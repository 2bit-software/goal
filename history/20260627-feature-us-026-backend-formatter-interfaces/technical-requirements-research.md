# Technical Requirements & Research — US-026

## Architecture context (REWRITE-ARCHITECTURE.md §3.1, §10 Phase 2)

This is the first Phase 2 story: it lands the back-end seams the rest of Phase 2
plugs into.

- `Backend` — `interface { Emit(*ast.File, *sema.Info) (pipeline.Output, error) }`.
  The Go transpiler, the future interpreter, and the future VM are
  implementations. This is the seam that makes the runtime a drop-in.
- `Formatter` — `interface { Format([]byte) ([]byte, error) }`. The Go backend
  uses `go/format`; the self-hosted build may shell out to `gofmt`.

## Existing seams to reuse

- `pipeline.Output{Go, Test string}` is the transpile output type referenced by
  the AC's `Output`. The new `Backend.Emit` returns `pipeline.Output`.
- `internal/parser.ParseFile(src) (*ast.File, error)` produces the AST.
- `corpus.RunCompile(root, Case, Transpiler)` is the behavioral tier: writes
  `Output.Go` into a temp module and runs `go build` + `go vet`. The AST engine
  satisfies `corpus.Transpiler` via `corpus.TranspilerFunc`.

## Package plan

- `internal/sema` (new): minimal `Info` struct (placeholder — US-027 fills it
  with enums/structs/signatures/from-registry/methods derived by AST walk). A
  `New()`/empty value is enough for US-026 to express `Backend.Emit`'s second
  parameter.
- `internal/backend` (new): defines `Backend` and `Formatter` interfaces; a
  `GoFormatter` implementing `Formatter` via `go/format`; a minimal AST
  `Backend` (`goBackend`) that emits Go source text for the plain-Go subset; and
  an engine entry point `Transpile(src string) (pipeline.Output, error)` that
  ties parser → sema → backend → formatter together (satisfies `corpus.Transpiler`).

## Import-cycle check

`internal/backend` imports `internal/ast`, `internal/parser`, `internal/sema`,
`internal/pipeline` (Output type only), `go/format`. None of those import
`backend`, so no cycle. `internal/sema` imports only `internal/ast`.

## Driver wiring

Add `--engine` to `parseFlags` in `cmd/goal/main.go` (default `splice`). When
`ast`, route per-file transpilation through `backend.Transpile`; otherwise keep
`pipeline.Transpile`/`pipeline.TranspilePackage`. Keep wiring minimal and
non-invasive so the splice path is byte-for-byte unchanged when the flag is
absent.

## Scope guard (avoid stepping on US-032)

The `goBackend` emitter implements only enough of the Go subset for a simple
plain-Go fixture (package clause, imports, `FuncDecl`, `GenDecl` var/const/type,
block/return/assign/expr statements, and the common expression/type forms it
needs). It returns a clear "unsupported node" error for goal-specific nodes
(enum/match/`?`/etc.) and any Go form not yet covered — those are US-032+ work.
