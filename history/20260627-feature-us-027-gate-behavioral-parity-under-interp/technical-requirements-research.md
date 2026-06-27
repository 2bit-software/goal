# Technical Requirements / Research — US-027

## Existing seams to reuse

- `internal/corpus.RunInterp(root, Case)` (US-025) runs ONE doctest case through
  the goscript interpreter in-process (no Go toolchain) and returns a
  case-identified error on any behavioral mismatch. This is the interpreter's
  entry into the implementation-independent behavioral conformance tier.
- `internal/corpus.Load(manifestPath)` + the committed `corpus/manifest.json`
  index every case. `manifestPath`/`repoRoot` are package-level test constants
  already used by the other gate tests.
- `TestInterpRunner` already drives every doctest case through RunInterp, and
  `TestASTEngineWholeCorpusBehavioralGate` (ast_gate_test.go) is the parallel
  Go-engine whole-corpus gate with the same loud-on-empty discipline.

## Approach

- New test file `internal/corpus/interp_gate_test.go` (package corpus).
- A package-test-level skip list `var interpGateSkips map[string]string`
  (case ID -> reason). Currently empty: all doctest cases pass under interp.
- The gate test:
  - Loads the manifest; fatals on empty.
  - Validates the skip list: every entry must have a non-blank reason and must
    name a real doctest case in the manifest (no stale/silent entries).
  - Iterates all doctest-kind cases; skips listed ones (logging the reason),
    runs the rest through RunInterp asserting no error.
  - Fatals if zero doctest cases ran (a narrowed/empty manifest cannot pass).
- A helper `blankSkipReasons(map[string]string) []string` returns the sorted IDs
  whose reason is blank, factored out so a focused unit test proves the gate
  fails when a skip entry lacks a reason.

## Constraints

- Zero-dependency: stdlib `testing` only (no testify).
- No production-code change expected — this composes existing seams.
- internal/corpus already imports internal/interp (no cycle).
