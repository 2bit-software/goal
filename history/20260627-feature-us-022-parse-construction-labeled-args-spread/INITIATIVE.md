# us-022-parse-construction-labeled-args-spread

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no feature branch — loop runs linear on base)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

US-022: Parse construction, labeled args, spread. Parser parses VariantLit with
LabeledArg arguments and SpreadElement (...defaults, ...derive(s)) inside
composite literals. Test parses Status.Active(since: now()) and a literal
containing ...defaults and asserts the nodes.
