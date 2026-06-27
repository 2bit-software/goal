# Verify — Quality

## Result: PASS

- Error handling matches the spec: a closed-E `?` with a non-closed-E callee fails
  with a descriptive message; a missing `from func` conversion across error types
  fails with the legacy wording. Both mirror internal/pass/closed.go.
- The encoding is read off sema.Info facts (FuncSig.{Mode,T,E}, FromRegistry), not
  token scans — correct by construction (no comma-split / mis-resolution class of
  bug).
- The behavioral tier (go build + go vet on the generated Go in a temp module) is
  the strong gate: it proves the lowered Go is well-typed, not merely substring-
  matched. All three closed-E shapes (match, `?` same-E, `?` cross-E) compile.
- No regressions: the full suite stays green; the closed-E match guard that
  previously failed in resultMatch now routes to closedResultMatch, and the open-E
  paths (03/04/05) are untouched.
- `from func` emission is scoped minimally (FuncFrom only); `derive func` still
  fails loudly, correctly deferred to US-039.

## Edge cases covered
- Arm with unused binding: closedResultMatch / unwrapClosed introduce the guard
  and the `binding := guard.Value` alias only when the binding is used (usesIdent),
  avoiding an unused-variable compile error.
- Bare/discard `?`: unwrapClosed emits `var _ T` and `_ = guard.Value` (valid Go).

## Notes
- Output is not byte-identical to the goldens (gensym `r` vs legacy `__goal_e`);
  this is expected and deferred to US-042 (golden regeneration). The transpile
  golden runner still compares the splice engine, which is unchanged.
