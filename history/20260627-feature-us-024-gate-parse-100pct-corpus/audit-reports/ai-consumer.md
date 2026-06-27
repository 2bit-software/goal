# AI-Consumer Readiness Audit — US-024

## Findings

No CRITICAL findings. No MAJOR findings.

The spec is implementable without guessing:
- Inputs are fully enumerated by `corpus/manifest.json` (107 cases; unique
  `.goal` paths derivable from file-mode `Input` + package `Files`).
- "Parse" maps to the existing `parser.ParseFile(src) (*ast.File, error)` API.
- Acceptance criteria are directly assertable: count failing inputs; require 0.
- The grammar gaps to close are concretely enumerated in
  `technical-requirements-research.md` (type-arg lists, type-literal operands,
  optional-colon payload fields) with example inputs.

### MINOR-1: Test placement not mandated by the spec
The spec is silent on where the gate test lives. Technical research recommends
`internal/corpus` (RunParse + Parser interface), consistent with the existing
runner pattern and free of import cycles. This is an implementation choice, not a
spec gap.

## Assumptions

- The gate test belongs in `internal/corpus` as a `Parser`-interface runner
  (`RunParse`), mirroring RunTranspile/RunCheck.
- New AST node `IndexListExpr` (parallel to go/ast) represents multi-element
  index/type-arg lists; single-element keeps the existing `IndexExpr`.
