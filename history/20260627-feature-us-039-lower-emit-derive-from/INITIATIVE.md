# us-039-lower-emit-derive-from

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no branch created — loop runs linear on the base branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | completed | 2026-06-27 |
| plan | completed | 2026-06-27 |
| tasks | completed | 2026-06-27 |
| implement | completed | 2026-06-27 |
| verify | in_progress | 2026-06-27 |

## Description

US-039: Lower and emit derive/from on the AST backend. The `from func` modifier
is already stripped (US-037 wired FuncFrom through funcDecl); this story adds
`derive func` lowering: expand each derive func field-by-field using the resolved
types in sema.Info (Structs + FromRegistry), not string parsing. The
12-derive-convert transpile cases (slice.goal, from_storage.goal, to_storage.goal)
must pass the behavioral tier (temp-module go build + go vet) through the new
backend.
