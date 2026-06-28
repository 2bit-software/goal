# US-006 rewire pipeline and check tests to corpus runner

**Type**: refactor
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop constraint)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

Rewire internal/pipeline/pipeline_test.go and internal/check/check_test.go to
delegate to the internal/corpus runner (RunTranspile / RunDoctest / RunCheck over
corpus/manifest.json) so that all hardcoded feature-dir lists and
filepath.Join("..","..","features",...) paths are removed and the full suite
stays green.
