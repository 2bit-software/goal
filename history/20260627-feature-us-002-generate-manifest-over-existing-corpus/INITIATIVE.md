# US-002 generate manifest over existing corpus

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch per loop constraint)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

Generate a corpus manifest over the existing golden files. A generator walks
features/NN/examples, testdata (top-level), and testdata/check, and writes
corpus/manifest.json without moving any source file. A test asserts the
generated manifest contains 51 transpile pairs and 50 check cases.
