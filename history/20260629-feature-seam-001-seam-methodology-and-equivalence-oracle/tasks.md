# Implementation Tasks — SEAM-001

## Task 1: Append the "Seam methodology" section to DECISIONS.md
- **Files**: `DECISIONS.md` (1 file)
- **Depends on**: none
- **Instructions**: Append a new section after the US-013 section (EOF) titled
  `## Seam methodology — equivalence oracle for cross-package idiom changes`,
  following the existing audit-section house style (blockquote intro + bulleted
  decision/procedure entries). Cover, per the spec:
  - the two gates contrasted (per-package byte-identical vs seam relaxed);
  - the crux (fixpoint proves stage1==stage2, not output==before);
  - the three equivalence proofs (fixpoint self-consistency, corpus behavioral
    tier, reviewed golden regeneration);
  - test classification (EXPECTED to change: go/ast-mirror units, golden
    transpile-shape fixtures; MUST stay green: task fixpoint, corpus
    behavioral/check tiers, full task check after golden regen);
  - the regenerate-and-review procedure with concrete commands
    (`-update-goldens`, `-update-snapshots`) and a reviewer checklist.
- **Verify**: `grep -n "Seam methodology" DECISIONS.md` returns the new heading;
  the section contains the three required parts (gate contrast, test
  classification, regeneration procedure).

## Task 2: Confirm the verify gates are green
- **Files**: none (verification only)
- **Depends on**: Task 1
- **Instructions**: Run the prd verifyCommands.
- **Verify**: `task check`, `task build`, `task fixpoint` all exit 0.
