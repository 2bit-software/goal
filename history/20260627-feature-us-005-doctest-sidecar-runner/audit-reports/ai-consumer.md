# AI-Consumer Readiness Audit — US-005 Doctest Sidecar Runner

## Findings

No CRITICAL or MAJOR findings. An implementer can proceed without guessing:

- Data model (Case, Kind, Normalize) already exists in internal/corpus.
- The doctest-case set is deterministically selectable (golden is an emitted
  `_test.go`: imports `"testing"`, contains `func Test`), matching exactly the
  4 feature-11 examples and excluding testdata/doctest_funcs.
- The runner contract mirrors the existing transpile/check runners
  (Run<Kind>(root, Case, <Interface>) error), reusing gofmt normalization.
- Acceptance criteria are concrete enough to write test assertions from.

## Assumptions

- The committed manifest is regenerated as part of this story so doctest cases
  are present for the runner test to consume.
- generate_test.go's incidental `other != 0` guard is relaxed to assert the
  doctest count, while the 51-transpile / 50-check assertions (US-002's stated
  criteria) are kept intact.
