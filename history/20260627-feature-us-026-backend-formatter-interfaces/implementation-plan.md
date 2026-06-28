# Implementation Plan — US-026 Add Backend and Formatter interfaces

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/sema/sema.go` | Minimal `Info` type (placeholder; US-027 populates it) so `Backend.Emit`'s second parameter is expressible. |
| `internal/sema/sema_test.go` | Smoke test: a zero `Info` is usable (constructor returns non-nil). |
| `internal/backend/backend.go` | `Backend` and `Formatter` interfaces; `GoFormatter` (go/format impl); `goBackend` (minimal AST→Go emitter); `Transpile(src)` engine entry. |
| `internal/backend/emit.go` | The minimal Go-source emitter (plain-Go subset) used by `goBackend.Emit`. |
| `internal/backend/backend_test.go` | Interface-existence + Formatter + `Transpile` happy-path; AC2 behavioral-tier test on a no-goal-constructs fixture via `corpus.RunCompile`. |
| `internal/backend/testdata/plain.goal` | The no-goal-constructs fixture for AC2. |

### Modified Files
| File | Changes |
|------|---------|
| `cmd/goal/main.go` | Add `--engine` flag to `parseFlags` (default `splice`; `ast` selects the new engine; unknown value → usage error). Route the per-file/package transpile through the selected engine. Keep splice path identical when absent. |
| `prd.json` | Set `US-026.passes = true` (only after green). |
| `progress.txt` | Append the US-026 entry + any reusable patterns. |

## Package Structure

```
internal/
  sema/
    sema.go          # type Info struct{...}; New() *Info
    sema_test.go
  backend/
    backend.go       # Backend, Formatter, GoFormatter, goBackend, Transpile
    emit.go          # emitFile/emitDecl/emitStmt/emitExpr (plain-Go subset)
    backend_test.go
    testdata/
      plain.goal
cmd/goal/main.go     # --engine flag + engine selection
```

## Dependency Graph

1. `internal/sema` — depends only on `internal/ast` (for future facts; minimal now).
2. `internal/backend` — depends on `internal/ast`, `internal/parser`,
   `internal/sema`, `internal/pipeline` (Output type), `go/format`.
3. `cmd/goal/main.go` wiring — depends on `internal/backend` + existing `pipeline`.
4. Tests — `internal/backend/backend_test.go` depends on `internal/corpus`
   (RunCompile) and lives in an EXTERNAL `backend_test` package (corpus imports
   pipeline/check; backend imports pipeline; corpus does NOT import backend, so
   backend's test may import corpus without a cycle — confirm at build time).

No circular dependencies: nothing in ast/parser/sema/pipeline/corpus imports
`internal/backend`.

## Interface Contracts

```go
// internal/sema
package sema
type Info struct{ /* populated by US-027: enums, structs, signatures, ... */ }
func New() *Info { return &Info{} }

// internal/backend
package backend
type Backend interface {
    Emit(file *ast.File, info *sema.Info) (pipeline.Output, error)
}
type Formatter interface {
    Format(src []byte) ([]byte, error)
}
type GoFormatter struct{}
func (GoFormatter) Format(src []byte) ([]byte, error) // wraps go/format.Source

// goBackend is the AST Go emitter (minimal plain-Go subset for US-026).
type goBackend struct{}
func (goBackend) Emit(file *ast.File, info *sema.Info) (pipeline.Output, error)

// Transpile is the AST engine entry: parse -> sema -> backend.Emit -> format.
// Satisfies corpus.Transpiler via corpus.TranspilerFunc.
func Transpile(src string) (pipeline.Output, error)
```

`Transpile` flow: `parser.ParseFile(src)` → `sema.New()` →
`goBackend{}.Emit(file, info)` → `GoFormatter{}.Format([]byte(out.Go))` → set
`out.Go` to the formatted bytes → return. Emitter errors (unsupported node)
propagate as a descriptive error.

## Integration Points

- `cmd/goal/main.go`:
  - `parseFlags` gains `engine string` return (default `"splice"`). Recognize
    `--engine=splice|ast`; reject other `--engine=...` and bare `--engine`.
  - `transpileAll` / the build/run/check entry takes the engine and, for `ast`,
    transpiles each package file through `backend.Transpile` assembling a
    `pipeline.PackageOutput` (one `GoFile` per source via existing `goName`);
    for `splice` it calls `pipeline.TranspilePackage` exactly as today.
  - Minimal, additive: when engine is `splice` (default/absent), code path is
    byte-for-byte today's.
- `internal/backend.Transpile` ↔ `corpus.RunCompile` via
  `corpus.TranspilerFunc(backend.Transpile)` in the AC2 test.

## Testing Strategy

- `internal/sema/sema_test.go`: assert `New()` non-nil (placeholder coverage).
- `internal/backend/backend_test.go` (package `backend_test`, external):
  - `TestFormatterFormatsGo`: `GoFormatter{}.Format` formats unformatted Go.
  - `TestBackendInterfaceSatisfied`: `var _ backend.Backend = ...`,
    `var _ backend.Formatter = backend.GoFormatter{}` (compile-time existence).
  - `TestASTEngineTranspilesPlainGo`: `backend.Transpile(plainSrc)` returns Go
    that gofmt-parses.
  - `TestASTEngineBehavioralTier` (AC2): build a `corpus.Case{Kind:KindTranspile,
    Input:"internal/backend/testdata/plain.goal", Mode:ModeFile}` and assert
    `corpus.RunCompile(root, c, corpus.TranspilerFunc(backend.Transpile)) == nil`.
    `-short`-skip (spawns the go toolchain). `root` is `../..`.
- `cmd/goal/main_test.go` (existing): add a small `parseFlags` engine test
  (default `splice`; `--engine=ast`; unknown value errors) if the existing test
  file structure makes it natural; otherwise a focused new test in the package.

## Requirement → file map

- FR-1 Backend → `internal/backend/backend.go`.
- FR-2 Formatter → `internal/backend/backend.go` (`Formatter` + `GoFormatter`).
- FR-3 sema.Info → `internal/sema/sema.go`.
- FR-4 AST engine → `internal/backend/backend.go` (`Transpile`) + `emit.go`.
- FR-5 flag → `cmd/goal/main.go` (`parseFlags`).
- FR-6 default unchanged → `cmd/goal/main.go` (splice path untouched).
- AC2 behavioral → `internal/backend/backend_test.go` + `testdata/plain.goal`.
