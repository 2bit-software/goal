# US-007 Port ast package to goal

**Type**: feature
**Created**: 2026-06-29
**Branch**: main (no-branch, loop linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-29 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Reimplement internal/ast as goal source under selfhost/ast so the AST node
definitions and Walk are goal-native. Imports the ported token package, drops
the reflection-driven dump.go (debug-only, off the compile path). The ported
package must transpile via the smoke gate, the generated Go must compile, and
the existing internal/ast tests must pass against the transpiled package.
