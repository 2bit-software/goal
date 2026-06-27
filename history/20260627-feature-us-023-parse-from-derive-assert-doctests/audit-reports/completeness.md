# Audit: Completeness — US-023

## Findings

- **MINOR** — FR-3 does not state where a doc-comment run that precedes a
  NON-function declaration (e.g. a `type`) attaches. Resolution: the corpus only
  has `///` before functions; such runs are treated as ordinary documentation
  and dropped. Not blocking.
- **MINOR** — The spec does not pin exact field names of the new AST nodes
  (deliberately — those are implementation detail, kept out of the spec). The
  acceptance criteria are expressed in behavioral terms ("carries the
  from-modifier", "doc node contains N doctests") which are testable regardless
  of field naming.
- No CRITICAL or MAJOR findings. Every acceptance criterion maps to a committed
  example input with a determinable AST shape, plus the standard build/vet/test
  gate.

## Assumptions

- Doc comments attach to the immediately-following function only.
- A doctest's expected output is the run of `///` lines after a `>>>` line up to
  the next `>>>` or the end of the doc run.
- Bodyless `derive func` is detected purely by the absence of a `{` after the
  signature (reusing the existing optional-body logic in `parseFuncDecl`).
