# Verification Report — SEAM-CAP-3c

## Gates (prd.json verifyCommands)
- `task check` — PASS (full `go vet` + `go test ./...`, incl. corpus behavioral tier and
  the selfhost port gate compiling selfhost/sema/foreign.goal).
- `task build` — PASS (bin/goal, bin/goalc built clean).
- `task fixpoint` — FIXPOINT OK (goal-c-1 == goal-c-2 byte-identical on the new sema source).

## Acceptance criteria → evidence
- AC-1 "propagate sibling-.goal sealed implementor set (extend the enums-only goal-source
  path)": goalForeignDecls now projects info.Sealed/info.SealedImpls; EnrichForeign merges
  them. Evidence: TestEnrichForeignProjectsSealedImplementors (registry contains shape.Node
  + `*shape.Lit`/`*shape.Neg`).
- AC-2 "fixed in BOTH internal/ and selfhost/": internal/sema/foreign.go +
  selfhost/sema/foreign.goal edited identically; port gate (TestPortedSemaPackage/
  Backend/Typecheck) green; fixpoint green.
- AC-3 "2+-package fixture proves cross-package sealed match transpiles + behaves
  identically in the real topology": TestCrossPackageGoalSealedMatchLowers (lowers to
  `.(type)` + `case *shape.Lit:`/`case *shape.Neg:`, no `Node_` enum path) and
  TestCrossPackageGoalSealedBehavesLikeSwitch (per-package transpile of shape + use, temp
  module build, run vs reference `switch x := n.(type)`).
- AC "non-exhaustive Error / complete clean": TestCrossPackageSealedMatchExhaustive (clean)
  and TestCrossPackageSealedMatchNonExhaustiveIsError (`non-exhaustive-match` Error naming
  `*shape.Neg`).
- AC-5 "gates green, corpus behavioral unchanged": see Gates above; fixtures are additive.

## Quality notes
- Scope held: only cross-package implementor-set propagation. No SEAM-004 seal/conversion.
- Backend lowering unchanged (pattern-shape dispatch already handled cross-package); the
  capability is purely sema-side propagation — confirmed by the lowering test passing.
- `.go` foreign path returns nil sealed by design (real build uses .goal siblings),
  documented inline and in DECISIONS.md.

## Result
All acceptance criteria satisfied. No failures.
