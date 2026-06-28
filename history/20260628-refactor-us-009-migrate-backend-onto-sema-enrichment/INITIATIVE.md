# us-009-migrate-backend-onto-sema-enrichment

**Type**: refactor
**Created**: 2026-06-28
**Branch**: main (no branch — loop runs on linear history)

## Status

| Step | Status | Updated |
|------|--------|---------|
| scaffold | done | 2026-06-28 |
| verify | done | 2026-06-28 |
| cutover | done | 2026-06-28 |
| cleanup | done | 2026-06-28 |
| done | done | 2026-06-28 |

## Description

Migrate internal/backend/package.go off internal/analyze. The package-mode
driver must build cross-file and foreign type facts from sema.ResolvePackage and
sema.EnrichForeign (from US-001) instead of analyze.BuildPackage +
analyze.EnrichForeign. internal/backend must import no internal/analyze symbols,
and the corpus behavioral conformance tier must still pass.
