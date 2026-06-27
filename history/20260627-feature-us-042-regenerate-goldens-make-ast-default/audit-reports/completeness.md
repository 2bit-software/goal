# Audit: Completeness — US-042

## Findings
- MINOR: Spec does not state what happens to a transpile case whose AST output
  differs from its old splice golden only in dropped `//` comments. Resolution:
  this is expected and acceptable — the regenerated golden IS the AST output by
  definition (FR-2). No action needed.
- MINOR: Spec does not enumerate which goldens change vs. stay identical. Not
  required for implementation; the regeneration is uniform across all cases.

No CRITICAL or MAJOR findings. Every FR is testable; ACs map to existing test
seams (parseFlags default, RunTranspile/RunDoctest exact tier, the US-041
behavioral gate, TestBootstrapGoldenMatches).

## Assumptions
- The exact-tier tests are switched from pipeline.Transpile to backend.Transpile
  (rather than kept on both engines), since regenerated AST goldens cannot match
  splice output. Splice retains behavioral-tier coverage.
- Doctest sidecar goldens regenerate from Output.Test; all other transpile
  goldens from Output.Go, keyed on the existing isDoctestSidecar classification.
