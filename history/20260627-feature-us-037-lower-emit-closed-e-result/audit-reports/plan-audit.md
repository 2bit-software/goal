# Plan Audit

## Findings

No CRITICAL or MAJOR findings. The plan maps each functional requirement to a
concrete change in internal/backend, with verified file paths, a valid build
order (lower.go facts -> emit.go dispatch -> tests), and concrete signatures. It
reuses the existing seams (fnKind, gensym, calleeSig/calleeMode, armBody,
bindingName, usesIdent) added by US-034/035/036.

- MINOR: `roResultClosed` enum placement — append after `roOption`; iota value is
  irrelevant since roKind is only compared, never serialized.
- MINOR: prelude placement relative to imports — the 06 inputs have no imports, so
  "before the first non-import decl" and "right after package" coincide; the
  non-import guard is defensive for future inputs.

## Assumptions

- Closed-E `match` is dispatched on the scrutinee callee's mode, not the enclosing
  function (matches legacy + golden).
- `from func` emits as a plain function in this story (its FromRegistry
  registration is done by sema.Resolve); `derive func` stays deferred to US-039.
- Judged by build+vet (corpus.RunCompile), not byte-exact goldens (US-042).
