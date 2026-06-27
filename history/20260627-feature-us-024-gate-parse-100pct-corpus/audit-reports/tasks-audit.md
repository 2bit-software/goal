# Tasks Audit — US-024

## Findings
No CRITICAL. No MAJOR.

- 4 tasks, dependency-ordered (ast → parser → corpus → verify), each
  independently committable and ≤3 files.
- Coverage: FR-1/2/3 → Task 3; FR-4 → Tasks 1-2; build/vet/test → Task 4.
- Every plan file inventory entry appears in a task.
- Each task has a concrete verify command.

### MINOR-1
Task 4 is verification-only (no files) but is retained as an explicit gate and
iteration loop for any residual parse gap. Acceptable.

## Assumptions
- The three known grammar gaps are the complete set; Task 4 iterates if a deeper
  category appears.
