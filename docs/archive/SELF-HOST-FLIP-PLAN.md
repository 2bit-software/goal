# Self-Host Flip Plan — COMPLETE

Status: **landed.** The self-host flip is done; this planning document is retained
only as a pointer. The full plan body lives in git history.

## Outcome

Goal source is now the compiler's only source. The goal-written compiler lives
under `internal/<pkg>` as canonical `.goal` source with committed, colocated
generated `.go` (emitted by `task generate`, drift-gated by
`task verify-generated`). The hand-written Go transpiler has been retired; a
clean checkout builds the committed generated Go (the B-commit bootstrap), and
the 3-stage bootstrap + byte-identical fixpoint (`task bootstrap`,
`task fixpoint`) prove the goal-sourced toolchain reproduces itself.

Test/dev infrastructure intentionally stays Go: `internal/corpus`,
`internal/byexample`, the `internal/selfhost` bootstrap/port harness, and
`cmd/{corpus-gen,build-playground}`. The archived Go reference compiler remains
reachable through git history.

## Where the decisions live

- `DECISIONS.md` — the adopted layout, bootstrap trust model, and the per-package
  port records (US-001 through US-022).
- `README.md`, `REWRITE-ARCHITECTURE.md`, `SELF-HOST-RESEARCH.md` — describe goal
  source as canonical with committed generated Go.
