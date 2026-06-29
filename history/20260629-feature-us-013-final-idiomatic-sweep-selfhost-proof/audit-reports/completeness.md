# Completeness Audit — US-013

## Findings

No CRITICAL findings. No MAJOR findings.

### MINOR-1: "corpus check tier" not separately runnable
The spec maps AC-3 (corpus tiers) onto `task check`. The corpus transpile/behavioral/
check tiers run as Go tests under `internal/corpus` and the goal-built-package behavioral
gates under `internal/selfhost`, all inside `task check`. There is no standalone corpus
command. This is accurate to the repo and not a blocker; AC verifiable via `task check`.

### MINOR-2: fixpoint already depends on bootstrap
`task fixpoint` has `deps: [bootstrap]`, so running it alone rebuilds stage-0 ->
goal-c-1 -> goal-c-2 and diffs. No separate build step needed for AC-3/FR-3. Noted for
clarity; not a gap.

## Assumptions

- "Auto-convertible propagation site" == a `goal fix` source rewrite (a `fixed`
  outcome / non-empty `-inplace` diff). `skipped` and `suggestion` are advisory and do
  NOT count, consistent with the per-package audit machine checks (US-005..US-012).
- Reconnaissance already established the tree is at the autofix fixed point and that
  the only undocumented flagged file is `selfhost/main.goal`. The story's remaining work
  is documentation + running the gates, not new conversions.
