# Audit: Completeness

## Findings

### MINOR — value-position coverage stated as out of scope
The spec scopes statement/return/var positions and excludes cross-package `:=`
inference. This is acceptable: the reported failure and the SEAM-002+ consumers
are statement/return-position matches. No action needed.

### MINOR — payload variants deferred
Field-set reconstruction for payload-bearing cross-package variants is deferred.
The real seam targets (FuncMod, ChanDir, Mode, Severity) are tag-only enums, so
a tag-only fixture proves the capability. Acceptable scope boundary, documented.

### MINOR — degradation on unresolved import
The spec states unresolved imports degrade non-fatally (matching foreign-struct
behavior). This is the existing, tested behavior of EnrichForeign — no new
surface. No action.

No CRITICAL or MAJOR findings. The failure is reproduced, the mechanism is
identified, and acceptance criteria are test-writable.

## Assumptions

- The fixture enum is tag-only (data-less variants); payload reconstruction is
  out of scope for this story.
- "Behaves identically to the equivalent switch" is verified by a package-mode
  corpus build plus a focused behavioral unit test, not a new corpus tier.
- selfhost/.goal mirror is updated in lockstep with internal/.go to keep
  fixpoint green and deliver the capability to the self-hosted compiler.
