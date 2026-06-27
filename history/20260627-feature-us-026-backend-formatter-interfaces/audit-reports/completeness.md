# Audit: Completeness — US-026

## Findings

### MINOR — Output type provenance
The spec references `Output` (FR-1) as "the existing transpile output". This is
`pipeline.Output{Go, Test}`. Not a blocker; called out so the implementer uses
the existing type rather than introducing a new one.

### MINOR — Fixture provenance for AC2
AC2 requires "a goal file using no goal-specific constructs". The spec does not
say whether this is a new fixture or an existing corpus case. Either is
acceptable; a new minimal fixture keeps the test self-contained. Not blocking.

### MINOR — Package-mode behavior under `--engine=ast`
Spec puts package-mode AST routing out of scope. The driver's `--engine=ast`
path therefore needs a defined behavior for multi-file packages (it may transpile
per file through the AST engine, or the seam may only be wired for the single
transpile entry while the driver flag selects the engine for the per-file path).
Implementer should pick the minimal honest wiring and keep splice as the package
default. Not blocking.

No CRITICAL or MAJOR findings. The spec is implementable as written.

## Assumptions

- `Output` == `pipeline.Output`.
- The behavioral tier means the `corpus.RunCompile` machinery (temp module +
  `go build` + `go vet`).
- The splice engine remains the product default for this story.
- The AST emitter only needs to cover the plain-Go subset its fixture exercises.
