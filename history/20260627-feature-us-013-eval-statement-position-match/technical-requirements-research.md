# Technical Requirements / Research — US-013

## Known seams (from progress.txt)

- A statement-position match parses to `*ast.ExprStmt{X: *ast.MatchExpr}` and
  must be intercepted in the statement dispatch (execStmt), NOT in evalExpr
  (value-position match is US-014).
- Match arms carry a `*ast.VariantPattern` (the destructuring form); enum
  construction (US-012) produces the `VariantVal(enum, tag, fields)`
  tagged-union the match reads via `v.Variant.Tag` / `v.Variant.Fields`.
- Match-arm payload binding introduces names into a child scope (mirror the
  backend's per-arm rename seam, but at eval time bind into a `scope.NewChild()`).
- The proven-exhaustive default arm is a loud panic ("unreachable") reusing the
  established `panicSignal` control sentinel (see US-010/US-019 notes).

## Notes

- Dependencies must stay clean for the US-022 gate: no go/types, internal/backend,
  internal/typecheck.
- Tests: stdlib `testing`, no testify. Drive a real parsed 02-match-shaped
  program through newInterp + evalFn helpers.
