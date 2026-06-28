# Tasks — US-027 Gate: behavioral parity under interp

## Task 1: Skip-list mechanism and its enforcement test (foundation) — completed

- Files: `internal/corpus/interp_gate_test.go` (new)
- Add `var interpGateSkips = map[string]string{}` (case ID -> justification),
  documented as the explicit skip list; currently empty.
- Add `func blankSkipReasons(skips map[string]string) []string` returning the
  sorted IDs whose reason is whitespace-blank.
- Add `TestInterpGateSkipListRejectsBlankReason` proving blankSkipReasons returns
  exactly the blank/whitespace IDs and nothing for a fully justified list.
- Spec coverage: FR-2 (explicit skip list), FR-4 (fail on unjustified skip).
- Depends on: nothing.

## Task 2: Whole-corpus interpreter behavioral gate — completed

- Files: `internal/corpus/interp_gate_test.go` (same new file)
- Add `TestInterpWholeCorpusBehavioralGate`:
  - Load `manifestPath`; fatal if empty.
  - Validate `interpGateSkips`: fatal/error on any blank reason (via
    blankSkipReasons) and on any entry not naming a real doctest case.
  - Iterate doctest-kind cases; skip listed ones (log reason); run the rest
    through `RunInterp(repoRoot, c)` asserting nil; subtest per case.
  - Fatal if zero applicable cases ran.
- Spec coverage: FR-1 (whole-corpus gate), FR-3 (fail on behavioral failure),
  FR-5 (loud on empty/narrowed + stale skip).
- Depends on: Task 1.

## Coverage check

- FR-1 -> Task 2; FR-2 -> Task 1; FR-3 -> Task 2; FR-4 -> Task 1; FR-5 -> Task 2.
- Plan file inventory (`internal/corpus/interp_gate_test.go`) -> Tasks 1 and 2.
