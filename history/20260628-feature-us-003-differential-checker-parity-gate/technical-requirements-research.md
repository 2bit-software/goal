# Technical Requirements / Research — US-003

## Existing seams (reuse, do not reinvent)

- `internal/corpus/check_runner.go` — `Checker` interface, `CheckerFunc`,
  `RunCheck`, `parseWantMarkers`. Cases come from the committed manifest
  (`../../corpus/manifest.json`); check cases are `Kind == KindCheck`.
- `internal/corpus/sema_checker.go` — `SemaCheck(src)` already adapts the AST
  checker to `[]check.Diagnostic` (parse → sema.Resolve → sema.Check → convert).
- Legacy checker: `check.Analyze(src) ([]check.Diagnostic, error)`.
- Both diagnostic streams are `[]check.Diagnostic` with `Pos` as a byte offset;
  line is `check.OffsetToPosition(src, d.Pos).Line`.

## Approach

- Add a differential test in `internal/corpus` that, for each `KindCheck` case
  under `testdata/check/**`, runs `check.Analyze` and `SemaCheck`, projects each
  finding to a (file, line, feature, code, severity) key, and diffs the sets.
- A small allowlist of documented divergences (keyed identically) is subtracted
  before comparison; each entry must be backed by a `DECISIONS.md` note.
- Discover the actual divergences empirically by running the comparison, then
  record them in `DECISIONS.md` and (for AST-fires/legacy-deferred cases) update
  the `// want` markers.

## Notes

- `SemaCheck` uses single-file `sema.Resolve` + `sema.Check` (no package/foreign
  enrichment), matching `check.Analyze`'s single-file `analyze.Build`. Keep the
  comparison single-file so neither side gets cross-file/foreign facts the other
  lacks.
- Loop policy: stay on `main`, no feature branch.
