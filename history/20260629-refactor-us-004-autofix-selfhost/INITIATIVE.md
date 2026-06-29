# us-004-autofix-selfhost

**Type**: refactor
**Created**: 2026-06-29
**Branch**: main (loop runs on base branch, no feature branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-29 |
| plan | completed | 2026-06-29 |
| tasks | completed | 2026-06-29 |
| implement | completed | 2026-06-29 |
| verify | in_progress | 2026-06-29 |

## Description

US-004 (self-host idiomatic PRD), step 2 of 3: run `goal fix --inplace` across all
selfhost/*.goal to mechanically convert `(T, error)` + manual `if err != nil`
propagation into `Result[T, error]` + `?`. Verify the autofix has reached a fixed
point (re-running changes nothing), and that `task check`, `task build`,
`task fixpoint`, and the corpus behavioral tier stay green. This dogfoods
`goal fix` on real compiler code; if the fixer miscompiles or breaks something,
the fix lands in internal/fix as part of this story.
