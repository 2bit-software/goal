# us-021-eval-derive-and-from

**Type**: feature
**Created**: 2026-06-28
**Branch**: (none — loop runs on the existing base branch ralph/ast-frontend-rewrite)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-28 |
| plan | completed | 2026-06-28 |
| tasks | completed | 2026-06-28 |
| implement | completed | 2026-06-28 |
| verify | in_progress | 2026-06-28 |

## Description

US-021 Eval derive and from: the goscript interpreter (internal/interp) must
evaluate `derive func` conversions field-by-field using resolved sema types
(sema.Info.Structs and FromRegistry), applying from-registry conversions for
bridged fields, mirroring the backend's derive lowering but producing runtime
Values. A unit test over a 12-derive-convert shape asserts a derived conversion
produces the expected target struct.
