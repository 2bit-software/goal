# AI-Consumer Readiness Audit — SEAM-003

## Findings

### MINOR-1: variant reference surface form
An implementer must know dataless variants are referenced as `Enum.Variant`
(e.g. `Status.Pending`), confirmed from features/01-enums/examples/status.goal,
and cross-package as `pkg.Enum.Variant` (e.g. `sema.Mode.ModeResultClosed`),
confirmed from the SEAM-002 parser.goal construction sites. Documented in
technical-requirements-research.md §3.

### MINOR-2: match arm exhaustiveness vs `_` rest-arm
The implementer must know full enumeration (no `_`) lowers to a panicking default
(nil faults) while a `_` arm lowers to a real Go default (nil-safe). This is
load-bearing for the String() method, where preserving the original
"Warning -> warning, else -> error" semantics maps to `match s { Severity.Warning
=> "warning"; _ => "error" }`. Captured in research §4-5.

## Assessment

An AI agent can implement this without guessing: every consumer site is
enumerated by file:line in technical-requirements-research.md, the conversion
pattern is the proven SEAM-002 precedent, and the acceptance criteria are
machine-checkable. No CRITICAL or MAJOR findings.

## Assumptions

- All FuncSig values stored in `info.FuncSignatures` will have Mode set after
  fixing foreign.goal:222 (resolve.goal's funcSig already sets it). The remaining
  `FuncSig{}` zero returns are ok=false and never Mode-matched.
- DECISIONS.md is the canonical place to record the conversion (superseding US-011).
