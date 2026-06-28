# Plan Coverage Audit — US-004

- FR-1 (AST drives lexical stage) -> main.go checkPackage swap to
  `sema.AnalyzePackageInDir`. Covered.
- FR-2 (output unchanged over corpus) -> new `TestCheckCorpusOutputUnchanged`
  driving `goal check` over a corpus KindCheck case (e.g.
  testdata/check/02-match/non_exhaustive_stmt.goal -> `[non-exhaustive-match]`
  Error, feature 02-match). Covered.
- FR-3 (depth stage + dedup) -> dedup loop preserved with sema positions;
  TestCheckDepthStageCatchesElidedLiteral guards it. Covered.
- FR-4 (exit/note) -> TestCheckCleanProgramPasses, TestCheckDepthNoteOmits
  GeneratedDump guard it; cmdCheck tally updated to sema.Error. Covered.

No CRITICAL/MAJOR. No scope creep (only the named files change).

## Assumptions
- A KindCheck corpus case exists with a stable non-divergent finding for the
  exact-match e2e assertion.
