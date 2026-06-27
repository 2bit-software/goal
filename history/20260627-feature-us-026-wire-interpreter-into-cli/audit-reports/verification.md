# Verification — US-026 Wire interpreter into the CLI

## Gates (prd.json verifyCommands)
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./... -count=1` — PASS (all packages ok; cmd/goal + internal/interp green)

## Acceptance criteria
- `goal run --engine=interp <file.goal>` runs func main, exit 0, prints expected
  stdout — PASS (`TestRunInterpEngineExecutesMain`: prints "green").
- Unknown `--engine` value rejected — PASS (`TestRunInterpUnknownEngineRejected`).
- Default `goal run` behavior unchanged — PASS (`TestRunExecutesMain` still green;
  `--engine` defaults to `ast`, existing path untouched).
- No-main program exits non-zero — PASS (`TestRunInterpNoMain`).

## Story acceptance (prd US-026)
- "cmd/goal executes a .goal program via the interpreter (e.g.
  `goal run --engine=interp <file>`), running func main and exiting 0 on
  success" — satisfied by `cmdRunInterp` + `parseRunFlags`.
- "A cmd/goal test runs a sample .goal program through the interpreter path and
  asserts exit code 0 and the expected stdout" — `TestRunInterpEngineExecutesMain`.

## Notes
- Interpreter path consumes the shared AST + sema front-end directly (parser →
  sema.Resolve → interp.New/Run), no Go transpilation — matches the goscript
  design (REWRITE-ARCHITECTURE §3.1).
- guideCommands updated to document `--engine=ast|interp`; AI-KNOWLEDGE-BOOTSTRAP.md
  regenerated so TestBootstrapGoldenMatches stays green.

Result: VERIFIED.
