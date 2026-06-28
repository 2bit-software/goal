# Verify — US-004

## Acceptance criteria

- AC1 "checkPackage uses the US-002 sema driver and no longer imports
  internal/check": PASS. `checkPackage` calls `sema.AnalyzePackageInDir`; the
  `goal/internal/check` import is removed; only doc-comment mentions of
  `check.Severity` remain (no symbol use). `grep internal/check cmd/goal/main.go`
  -> none.
- AC2 "typecheck depth stage and lexical/depth dedup preserved (suppress-by-
  (basename,line,feature))": PASS. The `suppress` map and dedupKey logic are
  unchanged; depth severity is converted with `sema.Severity(d.Severity)`.
  TestCheckDepthStageCatchesElidedLiteral still passes (depth finding suppresses
  the lexical misfire).
- AC3 "end-to-end test asserts `goal check` output over the corpus check cases
  is unchanged": PASS. New `TestCheckCorpusOutputUnchanged` drives a corpus
  KindCheck fixture through `goal check` and asserts the
  `[non-exhaustive-match]` Error naming `Status.Cancelled`.

## verifyCommands

- `task check` -> PASS (all packages ok).
- `task build` -> PASS (exit 0).

## Findings

None (no CRITICAL/MAJOR/MINOR). The US-003 parity gate provides the corpus-wide
guarantee; this story flips the live path to the already-proven AST checker.
