# Initiative: ai-guide-command

**Type**: feature
**Status**: in_progress
**Created**: 2026-06-25
**ID**: 6a3d5299-feature-ai-guide-command

## Steps

| Step | Profile | Status | Updated |
|------|---------|--------|--------|
| spec | feature | completed | 2026-06-25 09:20 |
| plan | plan | completed | 2026-06-25 09:35 |
| tasks | tasks | completed | 2026-06-25 09:45 |
| implement | implement | completed | 2026-06-25 10:30 |

## Description

Replace the hand-maintained `AI-KNOWLEDGE-BOOTSTRAP.md` with a `goal ai` subcommand
(aliased `goal --ai`) that emits a complete "how to write goal" guide to stdout as Markdown,
assembled at invocation time so it cannot drift from the real toolchain. Feature examples are
re-transpiled live; CLI commands/flags and the checker's diagnostic codes are derived from
code; authored prose is embedded. The old file becomes a generated, golden-tested artifact.

## Goals

- Single source of truth: the binary, not a hand-edited doc.
- Feature lowerings shown == exactly what this binary produces (live `pipeline.Transpile`).
- Toolchain + diagnostic catalog derived from code, guarded by drift tests.
- Reuse the existing `docs/by-example.md` + its parser (lift to a shared internal package).
- Preserve zero-dependency / stdlib-only constraint.

## Progress

- 2026-06-25 — Research + design decisions complete; spec, technical research, and research
  summary written. Four key decisions confirmed with user (liveness, source, surface, old
  file). Spec signed off; plan + technical-spec + tasks written; implementation complete.

## Completion

**Completed**: 2026-06-25
**Steps**: spec ✓ · plan ✓ · tasks ✓ · implement ✓ (all complete)

### Outcomes
- Feature: `goal ai` / `goal --ai` binary-sourced AI bootstrap guide — **Complete**.
  - Live re-transpilation of feature examples; CLI/diagnostics derived from code; authored
    prose embedded; `AI-KNOWLEDGE-BOOTSTRAP.md` is now a golden-tested generated artifact.
  - 18/18 tasks complete. `go test ./...` green, `go vet`/`gofmt` clean, zero new deps.
  - Drift guards negative-tested (golden + catalog). build-playground refactor output-identical.

### Notes
- Surfaced three real drifts the live approach fixes: stale `__gop_` prefix (live is `__goal_`),
  false "checker not started" prose, and the old starter's non-compiling doctest.
- No Linear ticket associated.
