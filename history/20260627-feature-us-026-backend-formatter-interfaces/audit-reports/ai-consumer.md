# Audit: AI-Consumer Readiness — US-026

## Findings

### MINOR — Interface method names not fully pinned
FR-1/FR-2 give signatures (`Emit(*ast.File, *sema.Info) (Output, error)`,
`Format([]byte) ([]byte, error)`) but not concrete type names for the
implementations. The architecture doc supplies the intent; the implementer is
free to name the Go formatter and AST backend. Specific enough to write tests
from. Not blocking.

### MINOR — Error message text for unknown engine
FR-5 requires "a clear usage error" for an unknown `--engine` value but does not
fix the exact wording. A test can assert the value appears in the error without
pinning exact text. Acceptable.

All acceptance criteria are verifiable:
- Interface existence → compile-time (a test using the interface compiles).
- Flag selection → drive `run([]string{...})` / parse flags and assert.
- Behavioral tier → `corpus.RunCompile` returns nil.
- Verify gates → the three prd commands.

No CRITICAL or MAJOR findings. An AI agent can implement this without guessing.

## Assumptions

- The engine entry point is a free function `backend.Transpile(src) (pipeline.Output, error)`
  adaptable to `corpus.Transpiler`.
- `sema.Info` may be an (initially) empty struct; later stories populate it.
- `--engine` is added to the existing `parseFlags` for build/run/check.
