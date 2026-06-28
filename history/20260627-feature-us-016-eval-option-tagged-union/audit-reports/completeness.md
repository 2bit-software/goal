# Audit — Completeness

## Findings

- MINOR: The spec does not state behavior for a value-position match over Option
  (`x := match opt {...}`). The existing match seam (US-014) already handles
  value position uniformly via `selectMatchArm`/`armScopeFor`, so this is covered
  by construction; calling it out explicitly would be polish.
- MINOR: The None arm binding is implicitly "binds nothing" (None carries no
  payload). This is correct and falls out of `payloadValue` returning ok=false;
  no spec change required.

No CRITICAL or MAJOR findings. Happy path (Some/None construction + match),
error case (bad ctor / wrong arity), and the no-`*T` constraint are all covered.

## Assumptions

- Option is a built-in type identity ("Option"), not a user enum in `info.Enums`
  — so `Option.None` needs its own selector guard distinct from the enum
  data-less path. (Confirmed against eval.go.)
- The Some payload field name is "value", matching `resultOkField`.
