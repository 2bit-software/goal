# US-027 Gate: behavioral parity under interp

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs linear on the working branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

Add a whole-corpus interpreter behavioral parity gate. A gate test runs every
applicable corpus case (the doctest-kind subset, which the goscript interpreter
runs behaviorally via corpus.RunInterp) through the interpreter with zero
behavioral failures. Any excluded case must be enumerated in an explicit,
justified skip list (case ID -> reason); the gate fails if a case fails
behaviorally or if any skip-list entry lacks a recorded reason. This is the
interpreter analogue of the Go AST engine's TestASTEngineWholeCorpusBehavioralGate.
