# US-001 define cap capability model

**Type**: feature
**Created**: 2026-06-27
**Branch**: ralph/ast-frontend-rewrite (no new branch — repo works linearly on this branch)

## Status

| Step | Status | Updated |
|------|--------|---------|
| spec | in_progress | 2026-06-27 |
| plan | pending | - |
| tasks | pending | - |
| implement | pending | - |
| verify | pending | - |

## Description

Define an explicit capability/authority model so goscript restricts power by what
the host grants, not by a different grammar. Add `internal/cap` with a Capability
enumeration (Stdout, Stdin, FileRead, FileWrite, Net, Concurrency, Time, Env) plus
a CapabilitySet with Has/Grant and GrantAll()/DenyAll() constructors, a unit test
asserting GrantAll().Has(c) is true and DenyAll().Has(c) is false for every
Capability, and docs/goscript/restriction-diff.md enumerating each capability and
whether goscript grants it by default.
