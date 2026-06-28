# Verify: Acceptance Coverage — US-026

Verify gates (prd.json verifyCommands), all green:
- `go build ./...` — OK
- `go vet ./...` — OK
- `go test ./... -count=1` — OK (every package `ok`, incl. internal/backend,
  internal/sema, cmd/goal)

## AC → evidence

| Acceptance criterion | Evidence |
|---|---|
| `Backend` interface with `Emit(*ast.File, *sema.Info) (Output, error)` exists | `internal/backend/backend.go` (`Backend`); referenced by `TestInterfacesExist`; exercised end-to-end by `Transpile`/`TestASTEngineTranspilesPlainGo`. |
| `Formatter` interface exists (+ Go impl) | `internal/backend/backend.go` (`Formatter`, `GoFormatter`); `TestInterfacesExist`, `TestGoFormatterFormats`. |
| Driver selects engine via flag (`--engine=ast`) | `cmd/goal/main.go` `parseFlags` (`--engine=splice|ast`); `TestParseFlagsEngine`, `TestBuildWithASTEngine` (end-to-end `goal build --engine=ast`). |
| Unknown `--engine` value → usage error | `parseFlags` returns an error naming the value; `TestParseFlagsEngine` (`--engine=bogus`, bare `--engine`). |
| A no-goal-constructs file transpiles through the new engine and compiles via the behavioral tier | `internal/backend/backend_test.go` `TestASTEngineBehavioralTier` — `corpus.RunCompile("../..", plain case, TranspilerFunc(backend.Transpile))` returns nil (temp-module `go build` + `go vet`). Fixture: `testdata/plain.goal`. |
| Verify gates green | run above. |

No acceptance criterion is uncovered.
