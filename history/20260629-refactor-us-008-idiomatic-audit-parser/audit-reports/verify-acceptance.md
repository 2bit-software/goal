# Verify: Acceptance Coverage — US-008

Spec acceptance criteria mapped to evidence:

- **AC-1 (Result/? where it fits; switch->match where it fits):** No
  intra-package conversion fits. Evidence: grep of all five `.goal` files —
  `func .*error` yields only `ParseFile` (exported, oracle-pinned) + the void
  `errorf` recorder; `err != nil`/`, err` yields only `ParseFile`'s
  `errors.Join`; no in-file `enum` declared. Refusals recorded in DECISIONS.md
  "self-host idiomatic audit — US-008 (parser)". PASS (recorded-decision branch).

- **AC-2 (goal fix reports no remaining auto-convertible propagation sites):**
  `goal fix selfhost/parser/*.goal` → no content diff; only stderr is the
  `parser.goal:57` result-sig SKIP on the exported `ParseFile` (a deliberately
  non-auto-convertible site). PASS.

- **AC-3 (parser tests pass against transpiled package; fixpoint green):**
  `task check` green — includes `internal/selfhost` port gate (BuildTranspiled +
  BuildAndTest transpiling `selfhost/parser` and running it against
  `internal/parser/parser_test.go`) and `internal/parser`. `task build` green.
  `task fixpoint` → FIXPOINT OK (byte-identical). PASS.

## Assumptions
- "Recorded refusal-with-reason + no source change" is a passing outcome for this
  AC family (precedent US-005/006/007; AC-1 escape hatch).
