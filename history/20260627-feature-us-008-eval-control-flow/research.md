# Research — US-008 Eval control flow

No external research required: the implementation pattern is already established
in this codebase. Findings are internal to internal/interp.

## Control-flow signalling pattern (established, reuse it)

The interpreter already models non-local control flow as Go `error` sentinels
threaded up through `execBlock`/`execStmt` and recovered at a boundary:

- `returnSignal{vals []Value}` implements `error`; raised by `execReturn`,
  recovered in `callFunc` via `errors.As`.

break/continue follow the SAME pattern (the US-007 progress note explicitly
anticipated this): introduce `breakSignal` and `continueSignal` sentinel error
types. A for loop recovers both (break => stop iterating; continue => skip to the
post clause). A switch recovers `breakSignal` only (continue is not a switch
concern and must propagate out to the enclosing loop). `returnSignal` always
propagates past loops/switch to the call boundary.

## Scoping

`execIf` already opens `scope.NewChild()` for init and a further child for the
taken branch. for/switch bodies and bare BlockStmt mirror this: each iteration of
a for body runs in a fresh child of the loop scope; the loop's Init binds in the
loop scope so it persists across iterations.

## Go semantics to honour

- for: three-clause (`for i:=0; i<n; i++ {}`), condition-only (`for c {}`), and
  infinite (`for {}` — nil Cond is always true). Cond must be bool.
- switch: tagged (`switch x { case a: }`) compares each case expr to the tag via
  Value.Equal; tagless (`switch { case cond: }`) takes the first truthy case.
  default runs when no case matches. Go switch cases do NOT fall through, so each
  selected clause runs then breaks implicitly.
- break exits nearest for/switch; continue advances nearest for's post.

## Out of scope (other stories)

- RangeStmt (for-range) => US-009 composite types.
- goto, fallthrough, labelled break/continue => remain descriptive refusals.

## Confidence: High — the seam and pattern are proven by US-004..US-007.
