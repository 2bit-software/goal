# Plan Buildability Audit — US-036

## Verdict: buildable.

Each component maps to a named function/seam that already exists:
- `matchStmt` (emit.go) default branch — add enum dispatch.
- `returnStmt` (emit.go) — add `return match` dispatch.
- `stmt` `*ast.DeclStmt` (emit.go) — add `var = match` dispatch.
- `selectorExpr` (emit.go) — add the binding field-export branch.
- helpers `matchQualifier`/`enumOf`/`usesIdent`/`gensym`/`renames` reused as-is.

## Sequencing is correct
Lowering helper first, then dispatch wiring, then the corpus fixture (+ count fix
+ golden), then the behavioral test. Verify gates run last.

## Test strategy is sufficient
Behavioral tier (build+vet over the 02-match + new case via `corpus.RunCompile`)
exercises every FR. The existing `TestTranspileRunner`/`TestGenerateCounts`
provide regression coverage for the manifest change.

## Assumptions
- `corpus.RunCompile` is the established behavioral-tier entry (US-032..035 used
  it); reused here.
- No new package is needed; all changes live in `internal/backend` + the corpus
  fixtures + one count-test line.
