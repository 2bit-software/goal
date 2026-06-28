# Cleanup notes

Removed old copies now that callers point at internal/textedit:
- internal/scan/scan.go: deleted Replacement, Splice, BaseType, IsLineStart,
  NextNewline, LeadIdent, IsIdent, SplitAssign, IsStmtKeyword. Repointed the
  remaining internal uses (CalleeKey, ScanFuncs -> textedit.IsIdent;
  IsBareQuestionStmt -> textedit.NextNewline/IsStmtKeyword). Dropped now-unused
  sort and unicode imports; added goal/internal/textedit; updated package doc.
- internal/analyze/spans.go: ZeroLit already removed in cutover; strings import dropped.
- Updated stale doc comments referencing scan./analyze. homes in
  backend/lower.go, interp/eval.go, fix/fix.go to name textedit.

Verification:
- `task check` (go vet + full go test ./...): all packages pass.
- `task build`: bin/goal and bin/goalc build.
- internal/textedit imports only sort, strings, unicode (go list) — no text/scanner.
- No live references to the moved scan symbols or analyze.ZeroLit remain (only
  updated doc comments).

Assumptions: "old code" = the original copies in scan/analyze; LeadIdent had no
callers (dead) but was still relocated per the story's explicit list.
