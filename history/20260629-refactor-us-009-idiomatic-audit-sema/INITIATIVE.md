# US-009 Idiomatic audit: sema

**Type**: refactor
**Created**: 2026-06-28
**Branch**: main (no branch — loop runs on linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-28 |
| plan | completed | 2026-06-28 |
| tasks | completed | 2026-06-28 |
| implement | completed | 2026-06-28 |
| verify | in_progress | 2026-06-28 |

## Description

Idiomatic audit of selfhost/sema (step 3 of the SELF-HOST IDIOMATIC plan). sema
is the first ported package with a REAL Result/?/(T,error) surface. Convert
genuinely-fallible package-internal helpers to Result/Option + `?` where
behavior-preserving and not requiring cross-package caller edits; express in-file
diagnostic/mode kinds as enums + match where they fit. Record genuine
refusals-with-reason in DECISIONS.md. Keep the US-003 verbatim self-host oracle
green: never change a public signature its tests pin; sema port tests must pass
against the transpiled package; task fixpoint must stay byte-identical.
