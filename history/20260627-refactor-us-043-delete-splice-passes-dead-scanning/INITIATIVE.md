# us-043-delete-splice-passes-dead-scanning

**Type**: refactor
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (loop constraint: no new branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| scaffold | done | 2026-06-27 |
| verify | done | 2026-06-27 |
| cutover | done | 2026-06-27 |
| cleanup | done | 2026-06-27 |
| done | in_progress | 2026-06-27 |

## Description

US-043: Delete the obsolete token-splice machinery so the project has a single
front-end (the AST backend, already the default since US-042). Remove
`internal/pass` (the splice lowering passes), the `internal/pipeline` splice
engine functions that drive them, and the `scan`/`analyze` symbols that become
dead once those are gone. Re-point every remaining splice-engine call site onto
the AST backend and remove the `--engine=splice` flag. Behavior on the AST
default path is unchanged; build, vet, and the full suite stay green.
