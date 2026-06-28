# Scaffold notes — US-010

## Additive work done in scaffold (old code untouched)
- Copied the foreign-package fixture from internal/analyze/testdata/extpkg/types.go
  to internal/sema/testdata/extpkg/types.go (doc comment reworded analyze -> sema).
  This lets sema's foreign + package tests stop reaching into ../analyze/testdata
  before analyze is deleted in cleanup.

## Why no other side-by-side code
This refactor is a deletion: sema already provides the replacements
(DirResolver/DefaultResolver from US-001; ResolvePackage/EnrichForeign from
US-001/002). No new production code is needed — cutover just repoints the few
remaining analyze references onto the existing sema API and rewrites the
differential parity tests to sema-only assertions, then cleanup deletes
internal/analyze and internal/scan.

## How to test
The relocated fixture is exercised by the rewritten internal/sema foreign and
package tests during cutover. Full gate: `task check` then `task build`.
