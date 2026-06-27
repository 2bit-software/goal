# Research Findings — US-026

Internal architecture change; zero-dependency (stdlib only). No external library
research required. Findings are grounded in the existing codebase.

## How transpilation works today (the splice engine)

- `pipeline.Transpile(src) (Output, error)` — single-file: `analyze.Build` →
  ordered splice passes → `go/format` once. `Output{Go, Test}`.
- `pipeline.TranspilePackage(*project.Package) (PackageOutput, error)` — package
  path used by the driver (`cmd/goal`): merged tables, one prelude per package.
- The driver (`cmd/goal/main.go`) discovers packages, calls `TranspilePackage`,
  and drives `go build/run/vet` via an overlay. `parseFlags` parses `--emit` and
  a single path.

## The AST path the new engine will use

- `parser.ParseFile(src) (*ast.File, error)` — already parses 100% of the corpus
  (US-024) into `goal/internal/ast`.
- `ast.File` exposes `Package`, `Imports`, `Decls`. Node set covers the Go subset
  plus goal constructs.

## Behavioral tier (the AC2 oracle)

- `corpus.RunCompile(root, Case, Transpiler)` writes `Output.Go` to a temp module
  and runs `go build` + `go vet`. `corpus.Transpiler` is `Transpile(src)
  (pipeline.Output, error)`; `corpus.TranspilerFunc` adapts a free function.
- The AST engine entry `backend.Transpile(src)` will satisfy `corpus.Transpiler`,
  so AC2's test is `corpus.RunCompile(root, case, corpus.TranspilerFunc(backend.Transpile))`
  over a tiny no-goal-constructs fixture.

## Interface shapes (from REWRITE-ARCHITECTURE.md §3.1)

- `Backend.Emit(*ast.File, *sema.Info) (pipeline.Output, error)`.
- `Formatter.Format([]byte) ([]byte, error)` — Go impl wraps `go/format.Source`.

## Decisions

1. New packages: `internal/sema` (minimal `Info`) and `internal/backend`
   (interfaces + `GoFormatter` + minimal `goBackend` + `Transpile` engine entry).
2. Emitter scope: only the plain-Go subset the fixture needs; error on
   goal-specific / not-yet-supported nodes. Full subset = US-032.
3. Driver: add `--engine` (default `splice`); `ast` routes per-file transpile
   through `backend.Transpile`. Splice path unchanged when flag absent.

**Confidence: High** — all seams already exist and are exercised by passing tests.
**Open questions:** none blocking. Package-mode through the AST engine is out of
scope (US-033+); the driver's `ast` engine handles the single-file/per-file path.
