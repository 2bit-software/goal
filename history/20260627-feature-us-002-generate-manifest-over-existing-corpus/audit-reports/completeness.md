# Audit: Completeness — US-002

## Findings

- MINOR: The spec does not state behavior when a `.goal` file lacks its sibling
  `.go.expected`. Resolution: the generator only indexes pairs where both files
  exist; unpaired files are skipped. The fixed counts (51/50) act as the guard.
- MINOR: Ordering is required ("deterministic") but the sort key is not named.
  Resolution: sort by repo-root-relative path; deterministic and diffable.

No CRITICAL or MAJOR findings. Requirements are testable; counts are pinned.

## Assumptions

- Feature 11-doctests example pairs are counted as transpile pairs (per prd
  notes: 40 feature examples include them), not as separate doctest cases.
- `testdata/check` is walked recursively; top-level `testdata` is walked
  non-recursively (its `check/` subtree is the checker corpus, not transpile).
