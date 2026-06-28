# Tasks Audit — US-026

5 tasks, each independently committable, ≤3 files, concrete verify command,
ordered by the plan's dependency graph (sema → backend → tests/driver → finalize).

- Coverage: every FR/AC maps to a task (FR-3→T1; FR-1/2/4→T2; AC1/AC2→T3;
  FR-5/6→T4; verify gates→T5). No orphan tasks.
- Buildability: codebase compiles after each task (T1 standalone; T2 compiles
  against T1; T3/T4 against T2). External test package for T3 avoids the corpus
  import cycle (verified).

No CRITICAL/MAJOR findings.

## Assumptions
- The minimal emitter + implementer-chosen fixture are co-designed so coverage
  suffices.
- `--engine` on `check` is accepted but inert for now (documented in T4).
