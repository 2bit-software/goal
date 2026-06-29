# Audit: AI-Consumer Readiness — SEAM-001

## Findings

### MINOR — name the concrete mechanisms
An AI implementing later seam stories needs the EXACT regeneration commands, not
just "regenerate goldens". The methodology section SHALL name them concretely:
`go test ./internal/corpus -run TestUpdateGoldens -update-goldens` for the
`.go.expected` exact-tier goldens, and `go test ./internal/parser -run <snapshot>
-update-snapshots` for the AST snapshots. These are confirmed present in the tree
(internal/corpus/update_goldens_test.go, internal/parser/snapshot_test.go).

### MINOR — reviewer checklist must be concrete
"What a reviewer checks" should be an explicit short checklist (diff is
gofmt-stable, behavior-preserving, fixpoint still byte-identical, corpus
behavioral still green) rather than a vague instruction, so the review step is
reproducible.

## Verdict

Implementable without further clarification. All terms (fixpoint, corpus tiers,
goldens, go/ast-mirror) are already defined by the existing DECISIONS.md audit
sections this section sits alongside. Acceptance criteria are verifiable by
reading the section and running task check / build / fixpoint.

## Assumptions

- The section's structure mirrors the existing audit sections (blockquote intro +
  bulleted decision entries) for consistency.
- The go/ast-mirror "expected to change" classification is forward-looking
  (it bites in SEAM-004), correctly documented now but not exercised in SEAM-001.
