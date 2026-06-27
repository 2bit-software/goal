# US-026 Wire interpreter into the CLI

**Type**: feature
**Created**: 2026-06-27
**Branch**: (none — runs on current branch per repo loop convention)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Let a goal user run a .goal program under the goscript tree-walking interpreter
from the command line: `goal run --engine=interp <file>` parses, sema-resolves,
and interprets the program, running `func main` and exiting 0 on success.
