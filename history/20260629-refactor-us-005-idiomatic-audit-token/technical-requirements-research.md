# Technical Requirements / Research — US-005

## Audit findings for selfhost/token/token.goal

- **Kind const block (iota):** `type Kind int` plus an iota const block with
  unexported `*_beg`/`*_end` range markers. The integer values are load-bearing:
  `Kind` is used as an array index into `kindNames [...]string`, and the
  predicates `IsLiteral`/`IsOperator`/`IsKeyword` rely on `beg < k && k < end`
  integer-range comparisons. A goal `enum` lowers to a sealed interface +
  per-variant struct + marker (DECISIONS.md §01-enums, §8.1) — it is NOT an
  ordered integer type, cannot be array-indexed, and supports no range
  arithmetic. Therefore the goal `enum` does NOT fit; record the deliberate
  decision in DECISIONS.md (the AC's explicit escape hatch).
- **switch-over-enum → match:** none. token.goal contains no `switch`
  statements (only the `SWITCH` keyword spelling string).
- **fallible helpers → Result/Option/`?`:** none returning `(T, error)`. The
  only multi-value helper is `Lookup(name string) (Kind, bool)` — the comma-ok
  idiom, pinned by the oracle test (`got, ok := Lookup(...)`). It is not a
  `goal fix` propagation site and changing its signature to `Option[Kind]`
  would break the behavioral oracle test, so it stays.
- **goal fix:** running `goal fix` over selfhost/token/token.goal produces no
  changes and reports nothing — criterion 2 already holds.

## Conclusion

The only source change required is the DECISIONS.md rationale entry recording
why `Kind` stays an iota const block. No behavioral change to token.goal.
