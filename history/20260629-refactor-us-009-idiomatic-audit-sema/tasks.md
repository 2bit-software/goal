# Tasks — US-009 Idiomatic audit: sema

Status: T1 completed, T2 completed (all gates green), T3 completed.

## T1 — Convert sema.Analyze to Result/? (foundation, no deps) — completed
- File: `selfhost/sema/analyze.goal`
- Change `Analyze` to return `Result[[]Diagnostic, error]`, propagate the parse
  step with `parser.ParseFile(src)?`, and return success via
  `return Result.Ok(Check(file, info))`. Update the doc comment accordingly.
- Spec coverage: FR-1.

## T2 — Verify behavior + fixpoint (depends on T1)
- Run `goal fix selfhost/sema/*.goal` (per file): expect no source diff; only
  documented skip/suggestion reports.
- Run `go test ./internal/selfhost -run TestPortedSemaPackage` (sema behavioral gate).
- Run `task check`, `task build`, `task fixpoint` (expect FIXPOINT OK, byte-identical).
- If any gate fails for the Analyze conversion, fall back: revert analyze.goal and
  document Analyze as a refusal too. (Risk mitigation.)
- Spec coverage: FR-3, all acceptance criteria.

## T3 — Record decisions + bookkeeping (depends on T2 green)
- File: `DECISIONS.md` — add "self-host idiomatic audit — US-009 (sema)" section:
  the one conversion (Analyze) plus every refusal-with-reason.
- File: `prd.json` — set US-009 `passes: true`.
- File: `progress.txt` — append the US-009 entry; add any reusable pattern to the
  Codebase Patterns block.
- Spec coverage: FR-2.

## Coverage check
- FR-1 -> T1. FR-2 -> T3. FR-3 + acceptance criteria -> T2.
- File inventory (analyze.goal, DECISIONS.md, prd.json, progress.txt) all covered.
- Ordering is a valid topological sort: T1 -> T2 -> T3.
