# Technical Requirements / Research — US-005

## Current state

- `internal/corpus` already has the Case model (`corpus.go`), the manifest
  generator (`generate.go`), the transpile runner (`runner.go`), and the check
  runner (`check_runner.go`).
- The committed manifest (`corpus/manifest.json`) currently has 51 transpile
  cases and 50 check cases and ZERO doctest cases.
- The 4 feature-11 examples (add/enum/mixed/multi) have `.go.expected` goldens
  that are doctest `_test.go` sidecars (they `import "testing"` and contain
  `func Test...`). They are currently indexed as transpile cases and pass the
  transpile runner only via its `Output.Test` fallback.
- `testdata/doctest_funcs.go.expected` has doctests in source but its golden is
  the MAIN Go output, so it is a genuine transpile case (not a sidecar).

## Approach

1. `generate.go`: additionally emit a `KindDoctest` case for each transpile
   pair whose golden is a doctest sidecar. Detect a sidecar by golden content:
   it imports `"testing"` AND contains `func Test`. This cleanly matches exactly
   the 4 feature-11 examples and excludes doctest_funcs. The existing transpile
   classification is kept (so US-002's 51/50 counts are unchanged); the doctest
   case is additive, sharing the same Input/Expected with Normalize=gofmt.
2. `doctest_runner.go`: `RunDoctest(root, Case, Transpiler) error` — reads the
   input, transpiles, gofmt-normalizes both `Output.Test` and the golden, and
   compares. Reuses `gofmtNormalize` from runner.go.
3. `doctest_runner_test.go`: `TestDoctestRunner` iterates KindDoctest cases from
   the committed manifest against `pipeline.Transpile`, failing loudly if zero.
4. Regenerate `corpus/manifest.json`.
5. Relax `generate_test.go`'s `other != 0` guard to instead assert the doctest
   count, while keeping the 51 transpile / 50 check assertions intact (US-002).
