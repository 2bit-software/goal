# Plan Audit: Buildability — US-009 sema

## Findings

- Dependency order valid: edit -> verify -> record. No forward references.
- Interface contract concrete: exact before/after of `Analyze` given in goal syntax;
  `Result.Ok(...)` construction and postfix `?` are already lowered by the frontend
  (confirmed in backend emit). ModeResult -> (T,error) lowering keeps the emitted
  signature, so the transpiled Go compiles unchanged.
- File paths verified: selfhost/sema/analyze.goal, DECISIONS.md, prd.json,
  progress.txt all exist.
- Integration points specific: Analyze has no callers/tests (grep-verified);
  parser.ParseFile returns (*ast.File, error) = valid open-E `?` source.

Risk: if the frontend cannot lower a `Result[[]Diagnostic, error]` return with a
slice T, the transpile/fixpoint would fail. Mitigation: the verify step runs
`task build` + `task fixpoint`; if it fails, fall back to pure-refusal (revert the
Analyze edit, document Analyze as a refusal too). This is the only live risk and it
is gated.

No CRITICAL or MAJOR findings.

## Assumptions

- The goal frontend lowers a named-generic `Result[SliceType, error]` return the
  same as any other open-E Result. To be confirmed empirically in verify.
