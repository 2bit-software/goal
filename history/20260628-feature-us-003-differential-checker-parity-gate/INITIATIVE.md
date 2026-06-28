# US-003 differential checker parity gate

**Type**: feature
**Created**: 2026-06-28
**Branch**: main (loop policy: no branch — linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-28 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Add a differential parity test proving the sema (AST) checker and the legacy
internal/check checker agree on every case under testdata/check/**, comparing
findings by (file, line, feature, code, severity). The test passes only when
findings are identical except for divergences explicitly recorded in
DECISIONS.md. Any divergence where the AST checker fires and the legacy checker
deferred is recorded in DECISIONS.md as a documented improvement and the
corresponding // want markers are updated to the sema behavior.
