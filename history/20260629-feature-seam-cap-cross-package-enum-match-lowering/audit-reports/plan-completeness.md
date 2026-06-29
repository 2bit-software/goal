# Plan Audit: Completeness

## Findings

### MINOR — corpus manifest key naming
The plan registers a new ModePackage case. The existing convention derives the
case id from the input path (`testdata-package-foreign-derive`). The new case id
should follow the same convention (`testdata-package-cross-pkg-enum`). Noted for
implementation; not blocking.

### MINOR — value/return position both exercised
The fixture exercises statement and return position. inferMatchType (`:=`) is
out of scope per spec; the plan respects that. Complete.

No CRITICAL or MAJOR findings. Every spec FR maps to a file:
FR-1 -> lower.go matchQualifier; FR-2 -> foreign.go reconstruction;
FR-3 -> case-label builder (unchanged) + fixtures; FR-4 -> additive-only,
verified by corpus behavioral tier staying green.

## Assumptions
- The foreign fixture is hand-written generated-form Go (the §8.1 encoding),
  not produced by transpiling a .goal file, mirroring `extpkg` fixture style.
- selfhost mirrors are updated to keep fixpoint green and deliver the capability
  to the self-hosted compiler, even though selfhost source does not yet use it.
