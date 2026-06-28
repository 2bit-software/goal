# Cleanup notes — US-010

## Removed
- internal/analyze/ (token-scan fact builder + foreign reader + testdata/extpkg).
- internal/scan/ (the text/scanner-based lexer: Lex, Token, Match*, ScanFuncs,
  CalleeKey, FirstBodyBrace, ParamsClose, TopLevelComma, IsBareQuestionStmt, ...).

## Cutover edits that made deletion safe
- internal/lsp/server.go + package_test.go: resolver type analyze.DirResolver/
  DefaultResolver -> sema.DirResolver/DefaultResolver (identical underlying
  func type; diagnostics.go already converted via sema.DirResolver(...)).
- internal/sema/resolve_test.go + foreign_test.go: rewritten from
  differential-vs-analyze parity tests to sema-only assertions.
- internal/sema/{foreign,package}_test.go: extpkg fixture repointed from
  ../analyze/testdata/extpkg to the relocated internal/sema/testdata/extpkg.
- internal/textedit/textedit.go: doc comment reworded to drop the literal
  "text/scanner" so the criterion-3 grep has no live-tree match.

## Verification (prd verifyCommands + AC greps)
- `task check`: all packages ok.
- `task build`: bin/goal + bin/goalc built clean.
- AC1: no live import of goal/internal/analyze; dir removed.
- AC2: scan lexer symbols gone (package deleted); remaining `scan.` matches are
  historical prose in code comments, not symbol usage.
- AC3: text/scanner appears only under attic/ and features/_cut/.

## Assumptions
- The differential parity tests' purpose (prove sema == analyze during
  migration) is obsolete once analyze is deleted, so they were converted to
  direct sema assertions over the same fixtures rather than removed — preserving
  coverage of the resolved facts.
- Comment text that names old scan/analyze symbols (explaining why the lexer is
  NOT on the path) is acceptable under AC2/AC3, which target live symbol usage
  and imports, not prose.
