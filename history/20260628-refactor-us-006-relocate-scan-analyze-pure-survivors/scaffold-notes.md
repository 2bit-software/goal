# Scaffold notes

## New files
- `internal/textedit/textedit.go` — Replacement, Splice, IsLineStart,
  NextNewline, LeadIdent, IsIdent, SplitAssign, IsStmtKeyword.
- `internal/textedit/types.go` — BaseType, ZeroLit (ZeroLit calls the local
  BaseType instead of scan.BaseType).

## Differences from old
- Functions are byte-identical copies of the originals in internal/scan and
  internal/analyze, except ZeroLit now calls the package-local BaseType.
- Package imports only sort, strings, unicode — no text/scanner.

## Coexistence
- Old copies in internal/scan/scan.go and internal/analyze/spans.go are still
  present at this stage; callers will be repointed and the old copies removed in
  the cutover/cleanup steps.

## Independent test
- `go build ./internal/textedit/ && go vet ./internal/textedit/` passes.
- Behavior is covered indirectly by the existing fix/typecheck/analyze/project/
  sourcemap test suites once callers are repointed.
