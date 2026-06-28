# Scope — US-005 Delete internal/check

## What is being refactored and why
internal/check is the legacy lexical (token-scanning) checker. As of US-004 it
is off the live `goal check` path (cmd/goal runs sema). Several other packages
still import it. This story migrates those consumers to sema / token helpers and
deletes internal/check entirely.

## Old code (consumers of internal/check)
- cmd/goalc/main.go: check.Analyze, check.Diagnostic, check.Error, check.HasErrors
- internal/guide/guide.go: check.AnalyzePackage, check.OffsetToPosition
- internal/lsp/{diagnostics,codeaction,symbols,semantictokens}.go +
  {diagnostics,package,semantictokens,definition}_test.go:
  check.Analyze, check.AnalyzePackageInDirWith, check.Diagnostic,
  check.OffsetToPosition, check.Warning, check.Error
- internal/typecheck/{typecheck,implements,mustuse,nozero}.go +
  {nozero,mustuse}_test.go: check.Severity, check.Error, check.Warning,
  check.OffsetToPosition
- internal/corpus/{check_runner,sema_checker}.go +
  {check_runner,parity,sema_checker,sema_fields,sema_question}_test.go:
  Checker interface over check.Diagnostic, CheckerFunc adapter, check.Analyze

## New code (replacements)
- sema.Analyze(src) ([]Diagnostic, error): single-file analogue of
  AnalyzePackageInDir (parse, Resolve, Check). New file internal/sema/analyze.go.
- sema.HasErrors(diags) bool.
- sema.Diagnostic.Render(filename) string (Line/Col already on Pos).
- token.OffsetToPosition(src, off) Pos: pure offset->Line/Col helper, new home
  for the deleted check.OffsetToPosition (leaf package, no new coupling).
- corpus.Checker / RunCheck operate on sema.Diagnostic; SemaCheck returns
  sema.Diagnostic directly; CheckerFunc adapter removed (RunCheck takes the
  SemaCheck func directly).
- typecheck.Diagnostic.Severity becomes sema.Severity; check.Error/Warning ->
  sema.Error/Warning; goalPosition uses a local lineCol helper (avoids the
  go/token import-name clash).
- lsp keeps analyze.DirResolver as Server.resolve (PRD note: lsp survives as an
  analyze resolver-type consumer); the sema call converts sema.DirResolver(s.resolve).

## What must NOT change
- `goal check` behavior (already on sema).
- lsp diagnostic/range output, symbol ranges, semantic tokens.
- typecheck depth-check findings and severities.
- corpus // want marker semantics.

## Deletions
- internal/check/ directory (incl. check_test.go, foreign_test.go).
- corpus/check_runner_test.go (tested the deleted legacy checker).
- corpus/parity_test.go (US-003 gate comparing legacy vs sema — moot once legacy
  is gone; documented in DECISIONS.md).
