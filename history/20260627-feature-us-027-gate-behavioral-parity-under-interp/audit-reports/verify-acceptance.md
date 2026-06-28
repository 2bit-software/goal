# Verify: Acceptance Coverage — US-027

Full suite green: `go build ./...`, `go vet ./...`, `go test ./... -count=1` all
pass (internal/corpus ok).

| Acceptance criterion | Evidence |
|---|---|
| Gate runs every applicable case through RunInterp, zero failures | TestInterpWholeCorpusBehavioralGate iterates all KindDoctest cases, RunInterp per case (4 green, 0 skipped) |
| Excluded cases in an explicit, justified skip list; none silently dropped | interpGateSkips map (ID -> reason); gate logs each skip's reason |
| Gate fails on behavioral failure | per-case t.Error on RunInterp error; RunInterp returns a case-identified error on mismatch (covered by TestInterpRunnerMutatedExpectedFails) |
| Gate fails if skip list missing a recorded reason | blankSkipReasons -> t.Errorf in the gate; TestInterpGateSkipListRejectsBlankReason proves blank/whitespace reasons are caught |
| Gate fails loudly if zero applicable cases ran | t.Fatalf on empty manifest and on ran==0; stale-skip check against manifest doctest IDs |
| verifyCommands stay green | build/vet/test all pass |

No acceptance criterion is unmapped.

## Findings
- No CRITICAL/MAJOR/MINOR findings.

## Assumptions
- The interpreter's behavioral conformance universe is the doctest subset
  (RunInterp's domain); non-doctest tiers are out of scope, not skipped.
- The skip list ships empty because all four doctest cases pass; the blank-reason
  enforcement is proven by a dedicated unit test rather than by a populated list.
