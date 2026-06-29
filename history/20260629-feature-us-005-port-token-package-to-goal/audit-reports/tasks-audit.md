# Tasks Audit — US-005

## Coverage
- Every plan file appears in a task: selfhost/token/token.goal (T1),
  internal/selfhost/selfhost.go (T2), internal/selfhost/port_test.go (T3).
- Every spec FR/AC traces to a task (FR-1/AC-1->T1; FR-3 enabler->T2;
  FR-2/FR-3/AC-2/AC-3->T3; AC-4->T4).

## Buildability
- Tasks are ordered by dependency (source -> helper -> test -> gates).
- Each task touches <=1 file and is independently committable; the codebase
  compiles after T2 (helper unused-but-valid) and after T3.
- Each task has a concrete verification step.

No CRITICAL or MAJOR findings.

## Assumptions
- The four tasks are small enough to land in a single implementation turn; they
  are committed together as one story commit (loop convention).
