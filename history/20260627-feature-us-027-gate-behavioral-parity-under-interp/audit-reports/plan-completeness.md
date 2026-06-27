# Plan Audit: Coverage — US-027

## Coverage map

- FR-1 (whole-corpus gate) -> TestInterpWholeCorpusBehavioralGate iterating all
  doctest cases through RunInterp.
- FR-2 (explicit skip list) -> interpGateSkips map.
- FR-3 (fail on behavioral failure) -> per-case t.Error on RunInterp error.
- FR-4 (fail on unjustified skip) -> blankSkipReasons + TestInterpGateSkipListRejectsBlankReason.
- FR-5 (loud on empty/narrowed + stale skip) -> empty-manifest fatal, zero-case
  fatal, stale-skip check against manifest doctest IDs.

No plan element lacks a backing requirement (no scope creep). No spec requirement
is unmapped.

## Findings

- No CRITICAL/MAJOR/MINOR findings.

## Assumptions

- The interpreter behavioral tier is the doctest subset; non-doctest tiers are
  out of scope rather than "skipped".
