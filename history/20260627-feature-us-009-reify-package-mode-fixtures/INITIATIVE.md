# US-009 Reify package-mode tests as fixtures

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

Turn the inline package sources in internal/pipeline/foreign_test.go and
internal/pipeline/pipeline_package_test.go into on-disk multi-file package
fixtures with a declared import map, index them as Mode=package cases in the
corpus manifest, and add a corpus runner that executes the package-mode cases.
