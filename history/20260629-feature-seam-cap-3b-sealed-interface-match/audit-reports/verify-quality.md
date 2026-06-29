# Verify — Quality — SEAM-CAP-3b

- Error handling: non-exhaustive sealed match → Error; unresolvable sealed type →
  Warning (deferral, no false reject) — both tested. The lowering's else-branch
  emits a panicking default for a proven-exhaustive match (mirrors enumMatch),
  unreachable by construction.
- Edge cases covered: exhaustive (no diag), missing implementor (Error names `*Neg`),
  `_` rest-arm (accepted), unresolved/cross-package type (Warning). Binding rename
  to the type-switch guard is exercised by the behavioral test (l.Val / g.Inner).
- Spec consistency: same-package scope only; no cross-package propagation added
  (verified — SealedImpls is built only from in-package `implements` clauses; a
  cross-package sealed type defers). The feature is additive; enum/Result/Option
  matches and the fixpoint are unchanged (FIXPOINT OK).
- Tests assert real behavior: the behavioral test compiles and RUNS the transpiled
  Go against a hand-written reference type-switch over three shapes (incl. nested
  recursion), not just string-matching.

## Findings
- MINOR: value-type (non-pointer) implementor patterns are representable but unproven
  — documented Out of Scope; the §8.1/go/ast implementor shape is pointer-based.

No CRITICAL/MAJOR.
