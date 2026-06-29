# SEAM-007 go-to-goal upgrade skill

**Type**: feature
**Created**: 2026-06-29
**Branch**: main (loop-runner: no branch — linear history on base)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-29 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Build a repeatable skill (in the project's skill location) that upgrades an
existing Go codebase to idiomatic goal, scoped to a SINGLE go.mod package OR a
SINGLE .go file. The skill encodes the full pipeline proven across SEAM-002..006:
scope guard -> rename (.go->.goal, package clause, build tags, reserved-word
collisions) -> autofix (`goal fix`: (T,error)+manual if-err -> Result/?) ->
manual idioms (iota->enum, type-switch over sealed scrutinee->sealed interface +
exhaustive match, method-on-would-be-enum->free label function, fallible API->
Result/? where pure-propagation) -> verify + DECISIONS-style report.
Dogfooded on a real example to prove buildable idiomatic goal output.
