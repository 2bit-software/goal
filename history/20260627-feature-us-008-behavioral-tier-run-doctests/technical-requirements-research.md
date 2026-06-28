# Technical Requirements / Research — US-008

## Existing seams to reuse

- `internal/corpus/behavior_runner.go` `RunCompile` — temp-module machinery
  (os.MkdirTemp, minimal `module goalcorpus` / `go 1.26` go.mod, write a .go
  file, exec `go <verb> ./...` with cmd.Dir set, CombinedOutput for diagnostics).
- `internal/corpus/doctest_runner.go` `RunDoctest` — selects KindDoctest cases;
  doctest cases carry Output.Go (main package) and Output.Test (sidecar).
- `Transpiler` / `TranspilerFunc` interface seam (runner.go) — no new interface
  needed.
- Test pattern: `behavior_runner_test.go` skips under -short (spawns go
  toolchain per case), iterates the manifest, loud zero-case `t.Fatalf`.

## Plan

- Add `RunDoctestExec(root, Case, Transpiler)` to behavior_runner.go (or a new
  file): transpile the doctest case, require a non-empty Output.Test, write
  go.mod + case.go (Output.Go) + case_test.go (Output.Test) into a temp module,
  run `go test ./...`. Doctest cases share Input with their transpile twin, so
  Output.Go is the real package under test and Output.Test is the doctest.
- Add a test that drives all KindDoctest cases through pipeline.Transpile,
  skipped under -short, with a loud zero-case guard.

## Notes

- The generated Go is stdlib-only, so the minimal module resolves offline.
- The package clause comes from the generated source (e.g. `package mathx`).
