# Tasks Audit — SEAM-003

## Findings

No CRITICAL or MAJOR.

- T1 is the foundation (no deps); T2-T4 each depend only on T1 and may proceed in
  any order; T5 depends on all. Valid topological order.
- Each task touches a bounded file set (T2 is the largest but is a single package's
  mechanical requalification + matches). The atomic-seam nature means the tree is
  red between T1 and T5 by design — documented and matches the SEAM-002 precedent.
- Every spec FR maps to a task: FR-1/FR-2 -> T1; FR-3 -> T2/T3/T4; FR-4 -> T2
  (foreign), T3 (calleeMode/construction); FR-5 -> T1 (iota removed); AC gates -> T5.

### MINOR-1
T5 should grep for residual patterns after edits to catch a missed site before
running the gates — already noted in the task body.

## Assumptions

- No new test files are required (selfhost-only conversion proven by existing port
  gates + fixpoint), consistent with SEAM-002.
- If a shared internal port-gated test references Mode/Severity iota constants, it
  is relocated per the SEAM-002 pattern (handled reactively in T5 if it surfaces).
