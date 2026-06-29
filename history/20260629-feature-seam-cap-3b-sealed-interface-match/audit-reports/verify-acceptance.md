# Verify — Acceptance Coverage — SEAM-CAP-3b

Full suite: `task check` green (all packages incl. internal/corpus behavioral tier
and internal/selfhost port gates), `task build` green, `task fixpoint` = FIXPOINT OK.

| Acceptance criterion | Evidence | Status |
|---|---|---|
| Implementor registry from `implements` clauses mapping sealed type → implementors | sema.Info.SealedImpls (sema.go), populated in resolve.go resolveTypeDecl, unioned in Merge; exercised by sema + backend tests | PASS |
| Parser accepts type-pattern arms; backend lowers to `case *T:` (new sealedMatch, distinct from enumMatch); sema exhaustiveness over registry | parser goal_match.go parseTypePattern; backend sealedMatch; sema checkOneSealedMatch; TestSealedMatchLowersToTypeSwitch asserts `.(type)` + `case *Lit:`/`case *Neg:` and NO `Node_` enum form | PASS |
| Fixed in BOTH internal/ and selfhost/ | 11 internal/ files + 9 selfhost/ files changed symmetrically; selfhost port gate (TestPorted{Sema,Backend,...}) green | PASS |
| Regression fixture: transpiles, behaves identically to type-switch, non-exhaustive is a sema error | TestSealedMatchBehavesLikeTypeSwitch (build+run vs reference `switch x:=n.(type)`); TestSealedMatchNonExhaustiveIsError; TestSealedMatchRestArmAccepted; TestSealedMatchUnresolvedDefers | PASS |
| task check, build, fixpoint green; corpus unchanged | all three gates green; corpus tier inside task check unchanged (additive feature) | PASS |

No acceptance criterion is unmapped.
