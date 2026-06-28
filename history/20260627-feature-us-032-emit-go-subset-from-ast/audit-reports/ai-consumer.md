# Audit 2: AI-Consumer Readiness — US-032

## Findings

### MINOR — Test assertion targets are implicit but discoverable
The acceptance criteria are phrased behaviorally ("transpiles to valid Go",
"builds and vets cleanly"). The concrete assertion seam (`backend.Transpile` +
`corpus.RunCompile` via `corpus.TranspilerFunc`) is established by US-026's
tests, so an implementer can write assertions without guessing. Non-blocking.

### MINOR — Switch emission shape unspecified (intentional)
The spec deliberately omits the emission algorithm (no file paths or code shape),
per the spec's no-implementation-detail rule. The implementer derives the shape
from the existing `ifStmt`/`forStmt` emitters and go/format normalization. This
is the correct division of concerns, not a gap.

## No CRITICAL or MAJOR findings

All terms (switch, case clause, behavioral tier, Formatter) are defined by the
codebase. Data formats are AST node types already declared in `internal/ast`.
Acceptance criteria are specific enough to write test assertions from.

## Assumptions
- Format-once means a single `go/format.Source` pass owned by the engine
  (`backend.Transpile`), so the emitter outputs token-correct but unformatted Go.
- The behavioral fixture lives under `internal/backend/testdata/` alongside the
  existing `plain.goal`, consistent with US-026.
