# US-011 Host-function bridge for stdlib

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — loop-runner linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Add a host-function registry to internal/interp resolving at least fmt.Sprintf,
fmt.Sprint, fmt.Println, fmt.Errorf, and errors.New to native Go
implementations. An unresolved imported call (an imported package symbol with no
registered shim) produces a located, named error rather than a silent nil.
