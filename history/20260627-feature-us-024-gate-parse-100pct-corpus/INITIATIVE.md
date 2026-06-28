# us-024-gate-parse-100pct-corpus

**Type**: feature
**Created**: 2026-06-26
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop runs on the existing branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-26 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

US-024 — Gate: parse 100% of corpus. Add a parser gate test that iterates every
`.goal` input in the corpus manifest and parses it with zero errors, failing
loudly listing any input that does not parse. Close the parser grammar gaps the
gate exposes so the parser is proven complete against the spec corpus.
