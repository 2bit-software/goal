# us-006-idiomatic-audit-lexer

**Type**: refactor
**Created**: 2026-06-29
**Branch**: main (loop runs linear history, no branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| scaffold | in_progress | 2026-06-29 |
| verify | pending | - |
| cutover | pending | - |
| cleanup | pending | - |
| done | pending | - |

## Description

US-006 — Idiomatic audit of selfhost/lexer (step 3 of the SELF-HOST IDIOMATIC
plan). Convert switch-over-in-file-enum to `match` and fallible helpers to
Result/Option + `?` where natural and behavior-preserving. The US-003 verbatim
selfhost is the behavioral ORACLE — never change a public signature its tests
pin; lexer tests must still pass against the transpiled package; task fixpoint
must stay byte-identical green. Convert where the goal idiom fits; record
refusals-with-reason in DECISIONS.md. Machine check: `goal fix` reports no
remaining auto-convertible propagation sites for lexer.
