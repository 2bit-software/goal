# Verify — Quality — US-004

## Findings

- The fix matches the package's stated philosophy ("never emit incorrect code"):
  `result-sig` now refuses conversions whose call sites it cannot prove safe
  rather than emitting a Warn-and-convert that breaks callers.
- Both refusal paths are reported as `result-sig` Skips with actionable messages
  (`exported X has callers fix cannot see`; `X is called where ? cannot apply`),
  so the per-package audits know exactly what remains.
- The three concrete miscompiles found by the dry run (Discover, Analyze,
  goListResolve) are all now skipped; verified in the stderr report.
- No behavior change to `fixPropagate`/`fixPropagateInit`/`reportCallSites`/match.
- Existing conversion behavior preserved: unexported functions with no in-file
  callers (cmd tests) and with a single collapsible-propagation call site
  (TestConvertTupleToResult) still convert.

No CRITICAL or MAJOR findings.

## Assumptions
- `willResult` over-approximates "will become Result" by treating every
  structural candidate as convertible; for this corpus it yields zero conversions
  regardless, and `task build`/`task fixpoint` are the backstop against any
  residual miscompile. This is no less safe than the prior unconditional
  conversion and strictly more conservative.
