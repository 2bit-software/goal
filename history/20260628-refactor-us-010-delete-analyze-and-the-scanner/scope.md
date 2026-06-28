# Scope — US-010 Delete analyze and the scanner

## What is being refactored and why
internal/analyze (token-scan fact builder) and internal/scan (the text/scanner
lexer) are the last lexer-based crutches. Every live consumer has already been
migrated to sema (US-007/008/009) except a couple of references. Remove both
packages so the AST front-end is the only path before self-host.

## Old code (current live consumers of analyze/scan)
- internal/lsp/server.go: uses analyze.DirResolver + analyze.DefaultResolver for
  the resolver type.
- internal/lsp/package_test.go: uses analyze.DirResolver.
- internal/sema/resolve_test.go: differential parity test vs analyze.Build.
- internal/sema/foreign_test.go: differential parity vs analyze.EnrichForeign,
  using fixture ../analyze/testdata/extpkg.
- internal/sema/package_test.go: uses fixture ../analyze/testdata/extpkg.
- internal/scan is imported only by internal/analyze itself.
- internal/textedit/textedit.go has a doc comment mentioning "text/scanner"
  (false positive for the criterion grep).

## New code (goals)
- lsp resolver type -> sema.DirResolver / sema.DefaultResolver.
- sema parity tests rewritten to assert sema's resolved values directly
  (analyze comparison is obsolete once analyze is deleted).
- extpkg fixture relocated to internal/sema/testdata/extpkg; sema tests repointed.
- internal/analyze and internal/scan directories deleted.
- textedit doc comment reworded to drop the literal "text/scanner".

## What must NOT change
- sema's resolved facts and EnrichForeign behavior (verified by the rewritten,
  now sema-only, assertions over the same fixtures).
- lsp behavior — only the resolver type's package changes (sema re-exports the
  identical DirResolver/DefaultResolver shape from US-001).
- All existing depth-check, backend conformance, project, and source-map tests.

## Acceptance criteria (from prd.json US-010)
1. internal/analyze removed; no live import outside attic/ and features/_cut/.
2. scan lexer symbols (Lex, Token, Match*, ScanFuncs, Func, CalleeKey,
   FirstBodyBrace, ParamsClose, TopLevelComma, MatchQualifier, MatchBodyBrace,
   IsBareQuestionStmt) removed from the live tree.
3. grep for text/scanner returns matches only under attic/ and features/_cut/.
