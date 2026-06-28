# Cutover notes

Swapped all real call sites to internal/textedit:
- fix/{match,fix,resultsig,propagate}.go: scan.Replacement/Splice -> textedit.*,
  analyze.ZeroLit -> textedit.ZeroLit. fix no longer imports scan or analyze.
- analyze/{analyze,methods,foreign}.go: scan.IsIdent/SplitAssign -> textedit.*
  (scan still imported for the lexer-based helpers that stay).
- analyze/spans.go: ZeroLit removed (now in textedit); dropped unused strings import.
- project/project.go, pipeline/sourcemap.go, typecheck/implements.go:
  scan.IsIdent/IsLineStart -> textedit.*.

interp/eval.go and backend/lower.go reference these only in comments (they own
local helpers) — no code change needed.

Old copies still live in internal/scan/scan.go; removed in cleanup.

`task check` (go vet + full go test ./...) passes.

Assumptions: call-site discovery via grep over the live tree (excluding attic/
and features/_cut/) is complete; the moved functions are pure (no text/scanner),
verified by building internal/textedit in isolation.
