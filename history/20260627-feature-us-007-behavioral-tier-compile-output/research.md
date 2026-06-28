# Research — US-007 behavioral tier compile output

## Summary

The corpus already has an interface-based transpile runner
(`internal/corpus/runner.go`: `Transpiler`, `TranspilerFunc`, `RunTranspile`).
US-007 adds a *behavioral* tier alongside the exact-match tier: rather than
comparing generated Go text to a golden, it proves the generated Go compiles.

## Key findings

- `pipeline.Output{ Go, Test string }` — single-file transpile yields `Output.Go`.
- `cmd/goal/main.go` already shells out to the Go toolchain
  (`runGo` -> `exec.Command("go", verb, ...)`, `writeOverlay` ->
  `os.MkdirTemp`). The behavioral tier reuses this pattern but with a
  self-contained temp module per case (simpler than `-overlay`, and judges each
  case in isolation).
- Generated Go is zero-dependency (stdlib imports only). A minimal `go.mod`
  (`module goalcorpus` + `go 1.26`) is enough for `go build`/`go vet` to resolve
  stdlib offline.
- Golden package names vary (app, status, shape, mathx, access, traffic). A
  single non-`main` package in its own dir builds fine; no `func main` needed.
- Test placement: external `corpus` test package (cwd = internal/corpus,
  repoRoot `../..`, manifest `../../corpus/manifest.json`), mirroring the
  existing runner tests. Guard with `testing.Short()` since each case spawns the
  Go toolchain.

## Confidence

High — the machinery and Output shape are established; this is an additive
runner + test following existing patterns.

## Open questions

None blocking.
