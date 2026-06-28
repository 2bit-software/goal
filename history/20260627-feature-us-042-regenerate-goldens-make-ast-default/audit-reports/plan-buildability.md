# Plan Audit: Buildability — US-042

- Dependency order valid: add regen test -> regenerate goldens -> switch exact
  tests -> flip default -> regen bootstrap -> verify. No forward references.
- Interface contracts agree: backend.Transpile has the same signature as
  pipeline.Transpile (func(string)(pipeline.Output,error)) and already satisfies
  corpus.TranspilerFunc (used in ast_gate_test.go), so the swap type-checks.
- File paths verified to exist (main.go, the 3 test files, generate.go's
  isDoctestSidecar). New file path is unused.
- Import direction safe: backend does not import corpus; corpus tests and the
  external pipeline_test may import backend without a cycle.
- No CRITICAL/MAJOR.

## Assumptions
- `-update-goldens` regeneration runs once locally during implement; the flag is
  retained (durable, like -update-snapshots) but default-off so normal runs compare.
