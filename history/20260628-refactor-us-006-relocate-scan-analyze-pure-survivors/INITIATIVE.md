# US-006 Relocate scan/analyze pure survivors

**Type**: refactor
**Created**: 2026-06-28
**Branch**: main (loop-runner: no branch, linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| scaffold | in_progress | 2026-06-28 |
| verify | pending | - |
| cutover | pending | - |
| cleanup | pending | - |
| done | pending | - |

## Description

Move the lexer-independent helpers out of internal/scan and internal/analyze
into a new internal/textedit package so the scanner and analyze can later be
deleted without losing pure utilities. ZeroLit moves alongside the type helper
BaseType. No behavior change.
