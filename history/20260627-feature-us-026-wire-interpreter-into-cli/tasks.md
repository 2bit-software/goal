# Implementation Tasks — US-026 Wire interpreter into the CLI

## Task 1: Add `--engine=interp` run path to cmd/goal
**Status**: completed
**Files**: `cmd/goal/main.go`
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2, FR-3, FR-4, FR-5
**Verify**: `go build ./...` && `go vet ./...`

### Instructions
- Import `goal/internal/interp`, `goal/internal/parser`, `goal/internal/sema`.
- Add run-flag parsing that recognizes `--engine=<value>` (default `ast`). Keep
  the existing `--emit[=dir]` and single-path handling. Reject any unknown
  flag and any engine value other than `ast`/`interp` with a descriptive error.
- In `run()`'s `case "run"`, when engine == `interp`, dispatch to a new
  `cmdRunInterp(path, out, errOut)`; otherwise keep the existing
  `cmdRun(root, emit, emitDir, ...)` path unchanged.
- `cmdRunInterp(file, out, errOut)`:
  - Require `file` to be a single `.goal` file (descriptive error if it is a
    directory or lacks the `.goal` extension).
  - `os.ReadFile` → `parser.ParseFile(string(src))` → `sema.Resolve(file)` →
    `interp.New(f, info, interp.WithStdout(out))` → `ip.Run()`. Wrap any error
    with the file path for context. Return nil on success.
- Document the `--engine[=interp]` flag on the `run` entry in `guideCommands`.
  After editing guideCommands, regenerate the golden:
  `go run ./cmd/goal ai > AI-KNOWLEDGE-BOOTSTRAP.md` (TestBootstrapGoldenMatches).

## Task 2: Tests for the interpreter run path
**Status**: completed
**Files**: `cmd/goal/main_test.go`
**Depends on**: Task 1
**Spec coverage**: FR-1, FR-2, FR-3, FR-5 + acceptance criteria
**Verify**: `go test ./cmd/goal -count=1`

### Instructions
- `TestRunInterpEngineExecutesMain`: write a sample `.goal` file (printing via
  `fmt.Println`) into a temp dir; `run([]string{"run", "--engine=interp",
  file}, &out, &errOut)` → nil error, `strings.TrimSpace(out.String())` equals
  the expected text. Mirror `TestRunExecutesMain`.
- `TestRunInterpUnknownEngineRejected`: `--engine=bogus` → non-nil error.
- `TestRunInterpNoMain`: a `.goal` file with no `func main` → non-nil error.
- Stdlib `testing` only (no testify).
