# Audit: Completeness — US-006

## Findings

No CRITICAL findings.
No MAJOR findings.

### MINOR-1: "grep returns nothing" is not fully specified
The acceptance criteria say grep returns nothing for hardcoded feature paths but
do not pin the exact pattern. Mitigation: the implementation will remove the
literal `features` path segment and the `filepath.Join("..","..","features",...)`
list entirely, and a manual grep for both `features` and the join pattern over
the two files will confirm zero hits. Non-blocking.

### MINOR-2: Coverage-parity is asserted by count, not by case identity
FR-3 requires no coverage regression. The rewrite drives the manifest, which
US-002/US-005 already proved indexes the same 51 transpile + 4 doctest + 50 check
cases the legacy harnesses walked. Parity is therefore guaranteed by the manifest
contents plus the loud zero-case guard, not by per-case name mapping. Acceptable.

## Assumptions

- The corpus manifest is the agreed single source of truth for case enumeration
  (established by US-002).
- External `_test` packages are an acceptable house pattern (required to avoid the
  corpus->pipeline / corpus->check import cycle).
- `TestRegistryRuns` (checker spine smoke test) is worth preserving and will be
  carried into the rewritten file rather than deleted.
