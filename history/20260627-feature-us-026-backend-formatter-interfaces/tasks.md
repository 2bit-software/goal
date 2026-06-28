# Implementation Tasks — US-026

## Task 1: Add minimal `internal/sema` Info type
**Status**: pending
**Files**: `internal/sema/sema.go`, `internal/sema/sema_test.go`
**Depends on**: (none)
**Spec coverage**: FR-3
**Verify**: `go build ./internal/sema/... && go test ./internal/sema/... -count=1`

### Instructions
- New package `sema`. Define `type Info struct{}` with a doc comment noting it is
  a placeholder that US-027 populates with name-keyed facts (enums, structs,
  signatures, from-registry, methods) derived by AST walk.
- `func New() *Info { return &Info{} }`.
- Import nothing beyond what's needed (none for now). Keep it at the bottom of the
  dep graph.
- Test: `TestNewReturnsInfo` asserts `sema.New() != nil`.

## Task 2: Add `internal/backend` interfaces, GoFormatter, AST engine
**Status**: pending
**Files**: `internal/backend/backend.go`, `internal/backend/emit.go`
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-2, FR-4
**Verify**: `go build ./internal/backend/...`

### Instructions
- New package `backend`. Imports: `goal/internal/ast`, `goal/internal/parser`,
  `goal/internal/sema`, `goal/internal/pipeline`, `go/format`, `fmt`.
- Define interfaces:
  - `Backend interface { Emit(file *ast.File, info *sema.Info) (pipeline.Output, error) }`
  - `Formatter interface { Format(src []byte) ([]byte, error) }`
- `type GoFormatter struct{}`; `func (GoFormatter) Format(src []byte) ([]byte, error)`
  wraps `format.Source`.
- `type goBackend struct{}` implementing `Backend`. `Emit` walks the `*ast.File`
  and produces Go source text into `pipeline.Output{Go: ...}` (Test empty for the
  plain-Go subset). Use `emit.go` for the recursive emitter.
- `emit.go`: a minimal recursive Go-source emitter (string builder) covering the
  plain-Go subset: File (package clause + imports + decls), `GenDecl`
  (import/const/var/type specs), `FuncDecl` (name, params, results, body),
  `BlockStmt`, `ReturnStmt`, `AssignStmt`, `ExprStmt`, `IfStmt`(optional),
  and exprs: `Ident`, `BasicLit`, `BinaryExpr`, `UnaryExpr`, `ParenExpr`,
  `CallExpr`, `SelectorExpr`, `StarExpr`, and the basic types
  (`Ident` type, `ArrayType`, `MapType`, pointer). For any node it does not
  handle (incl. all goal-specific nodes: EnumDecl, MatchExpr, UnwrapExpr,
  VariantLit, SpreadElement, AssertStmt, FuncMod != FuncPlain, etc.), return a
  descriptive `fmt.Errorf("backend: unsupported node %T", n)` — these are US-032+.
  Do NOT attempt perfect formatting; `GoFormatter` normalizes afterward, so the
  emitter only needs gofmt-parseable output (valid tokens + balanced braces).
- `func Transpile(src string) (pipeline.Output, error)`: parse via
  `parser.ParseFile(src)`; on error return it. `info := sema.New()`. `out, err :=
  goBackend{}.Emit(file, info)`. Format `out.Go` via `GoFormatter{}.Format`; on
  format error wrap with the unformatted Go (mirror pipeline's error style). Set
  `out.Go` to the formatted text and return. This satisfies `corpus.Transpiler`
  via `corpus.TranspilerFunc`.

## Task 3: Fixture + backend tests (incl. AC2 behavioral tier)
**Status**: pending
**Files**: `internal/backend/testdata/plain.goal`, `internal/backend/backend_test.go`
**Depends on**: Task 2
**Spec coverage**: AC1 (existence), AC2 (behavioral tier)
**Verify**: `go test ./internal/backend/... -count=1`

### Instructions
- `testdata/plain.goal`: a small goal file using NO goal-specific constructs —
  e.g. `package mathx` with `func Add(a int, b int) int { return a + b }` and a
  second func calling it. Must be valid Go after transpile so `go build`/`go vet`
  pass.
- `backend_test.go` in EXTERNAL package `backend_test` (it imports
  `goal/internal/corpus`, which does not import backend → no cycle). Tests:
  - `TestInterfacesExist`: `var _ backend.Formatter = backend.GoFormatter{}`
    (compile-time existence of Formatter; Backend existence is exercised by
    `Transpile`).
  - `TestGoFormatterFormats`: feed unformatted Go, assert it formats (no error,
    output gofmt-stable).
  - `TestASTEngineTranspilesPlainGo`: `backend.Transpile(<plain src>)` → no error,
    `Output.Go` parses via `go/format.Source`.
  - `TestASTEngineBehavioralTier` (AC2): skip under `-short`. Build
    `corpus.Case{ID:"plain", Kind:corpus.KindTranspile, Mode:corpus.ModeFile,
    Input:"internal/backend/testdata/plain.goal"}` and assert
    `corpus.RunCompile("../..", c, corpus.TranspilerFunc(backend.Transpile))` is nil.

## Task 4: Wire `--engine` flag into the driver
**Status**: pending
**Files**: `cmd/goal/main.go`, `cmd/goal/main_test.go`
**Depends on**: Task 2
**Spec coverage**: FR-5, FR-6
**Verify**: `go build ./cmd/goal/... && go test ./cmd/goal/... -count=1`

### Instructions
- In `parseFlags`, add an `engine string` return value (default `"splice"`).
  Recognize `--engine=splice` and `--engine=ast`; for any other `--engine=...`
  value or a bare `--engine` (no value), return a usage error naming the offending
  value. Thread `engine` through `cmdBuild`/`cmdRun`/`cmdCheck` and `transpileAll`.
- `transpileAll(root, engine)`: when `engine == "ast"`, transpile each discovered
  package file through `backend.Transpile` and assemble a `pipeline.PackageOutput`
  (one `pipeline.GoFile{Name: goName(f.Name), Go: out.Go}` per source, plus
  `Tests` when `out.Test != ""`). When `engine == "splice"` (default), call
  `pipeline.TranspilePackage(pkg)` exactly as today (byte-for-byte unchanged).
  Keep `goName`/`testName` reuse. Note: the AST package path is best-effort for
  this story (single-file/plain subset); package-level prelude/cross-file is
  US-033+ and out of scope — the splice engine remains the default.
- `check` does not transpile via `transpileAll`; for `--engine` on check, accept
  the flag (so usage is uniform) but checking still uses the existing checker —
  the engine flag has no checker effect yet (note this in a comment).
- Test (`main_test.go`): `TestParseFlagsEngine` — default is `splice`;
  `--engine=ast` yields `ast`; `--engine=bogus` returns an error mentioning the
  value. Follow the existing `parseFlags` test style in the package.

## Task 5: Verify, finalize, log
**Status**: pending
**Files**: `prd.json`, `progress.txt`
**Depends on**: Tasks 1-4
**Spec coverage**: all (verify gates)
**Verify**: `go build ./... && go vet ./... && go test ./... -count=1`
**Instructions**: Run the full prd verifyCommands. Only when green: set
`US-026.passes=true` in `prd.json` and append the progress.txt entry. (The
morse-code complete step handles the commit.)
