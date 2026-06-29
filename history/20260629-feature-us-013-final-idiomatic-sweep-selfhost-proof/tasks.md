# Tasks — US-013 Final idiomatic sweep and self-host proof

Status: T-1 completed, T-2 completed, T-3 completed, T-4 in progress (finalize).

## T-1: Re-confirm the whole-tree autofix fixed point (foundation, no deps)
Copy `selfhost/` to a scratch dir, run `goal fix -inplace` on every `*.goal`, and
`diff -r` against the original. Expect an empty diff and a stderr report containing only
`skipped`/`suggestion` lines (no `fixed`). Output: the confirmed AC-1 evidence.

## T-2: Document deliberately-Go constructs in DECISIONS.md (depends on T-1)
Append a "self-host idiomatic audit — US-013 (final whole-tree sweep + self-host proof)"
section to `DECISIONS.md`: record the whole-tree proof, roll up the per-package refusal
documentation (US-005..US-012), and add the `selfhost/main.goal` `run`/`emitPackage`
bare-error refusals (the one file never given a per-package audit).

## T-3: Run the project gates (depends on T-2)
Run, in prd order: `task check`, `task build`, `task fixpoint`. All must be green;
`task fixpoint` must print `FIXPOINT OK` (goal-c-1 == goal-c-2 byte-identical). This also
exercises the corpus transpile/behavioral/check tiers and the selfhost behavioral port
gates (FR-3, FR-4).

## T-4: Finalize (depends on T-3)
Set `US-013.passes = true` in `prd.json`, append the progress.txt entry, and commit.
