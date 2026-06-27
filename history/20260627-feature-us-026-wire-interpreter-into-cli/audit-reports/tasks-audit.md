# Tasks Audit — US-026

- Two tasks, correctly ordered (Task 2 depends on Task 1).
- Each task lists files, spec coverage, and a concrete verify command.
- All FRs covered: FR-1..FR-5 traced to Task 1; acceptance criteria traced to
  Task 2 tests.
- Task 1 flags the golden regeneration gotcha (TestBootstrapGoldenMatches) when
  editing guideCommands — a known repo pattern, prevents a CI surprise.
- Verify commands match the prd verifyCommands (build, vet, test).

No CRITICAL or MAJOR findings.

## Assumptions
- Engine values limited to `ast` (default) and `interp`; others rejected.
