# Verify — acceptance criteria

All acceptance criteria met (commit d515744):

- [x] 2-level sealed hierarchy transpiles and `go build`s cleanly —
      TestNestedSealedBuildsAndBehaves builds the transpiled package in a throwaway
      `module goal`; a clean build proves T satisfies both Expr and Node.
- [x] `match` over A (Node) includes T and `match` over B (Expr) includes T; both
      equal the reference type-switch — TestNestedEvalMatchesReference (run inside
      the temp module) compares evalExpr/evalNode against refExpr/refNode for every
      value, including an Expr value fed to evalNode (cascade satisfies Node).
- [x] Non-exhaustive match over A and over B each a sema error —
      TestNestedMatchNonExhaustiveEmbeddingLevel (Expr omitting *Neg) and
      TestNestedMatchNonExhaustiveEmbeddedLevel (Node omitting the cascaded *Neg)
      both assert code `non-exhaustive-match`.
- [x] Existing flat sealed interfaces unaffected — full corpus + sealed_match +
      sealed_methods + crosspkg_sealed suites green under task check; fixpoint
      byte-identical.
- [x] Fixed in both internal/ and selfhost/ — mirrored; selfhost port gate
      (TestPortedSemaPackage/Backend/Typecheck) green.
- [x] task check, task build, task fixpoint all green (FIXPOINT OK); corpus
      behavioral unchanged (additive tests only).

## Assumptions
- The proof is same-package (Node/Expr + implementors co-located), matching
  SEAM-004's actual AST topology; cross-package nesting rides the existing CAP-3c
  foreign projection without change.
