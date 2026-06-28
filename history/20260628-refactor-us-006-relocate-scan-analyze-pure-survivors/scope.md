# Scope — US-006 Relocate scan/analyze pure survivors

## What is being refactored and why
The lexer-dependent passes (scan, analyze) are slated for deletion in later
stories (US-010). Several pure, lexer-independent helpers live inside them.
Relocate those survivors into a new honest package `internal/textedit` so the
scanner and analyze can later be removed without losing the utilities.

## Old code
- `internal/analyze/spans.go`: `ZeroLit(typ, decls, depth)` (uses strings + scan.BaseType).
- `internal/scan/scan.go` pure helpers (no text/scanner, no Token dependency):
  `Replacement` (type), `Splice`, `BaseType`, `IsIdent`, `IsLineStart`,
  `NextNewline`, `LeadIdent`, `SplitAssign`, `IsStmtKeyword`.

## New code
- New package `internal/textedit` holding all of the above. ZeroLit lives here
  with BaseType (the type helper it depends on). The package imports only
  `sort`, `strings`, `unicode` — never `text/scanner`.

## Callers to repoint
- ZeroLit (real calls): internal/fix/resultsig.go, internal/fix/propagate.go.
- Replacement/Splice: internal/fix/{match,resultsig,propagate,fix}.go.
- IsIdent: internal/pipeline/sourcemap.go, internal/project/project.go,
  internal/typecheck/implements.go, internal/analyze/{analyze,methods,foreign}.go.
- IsLineStart: internal/pipeline/sourcemap.go.
- SplitAssign: internal/analyze/{methods,foreign}.go.
- Internal scan.go references (CalleeKey, ScanFuncs, IsBareQuestionStmt) repoint
  to textedit.IsIdent / NextNewline / IsStmtKeyword.

## What must NOT change
- Behavior of fix, typecheck, analyze, project, sourcemap, scan — all existing
  tests must pass. The scanner-based functions (Lex, Token, Match*, ScanFuncs,
  CalleeKey, etc.) stay in internal/scan.
- internal/textedit and ZeroLit's home must not import text/scanner.
