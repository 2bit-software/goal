# Scaffold notes

New, additive code (old internal/check untouched at this step):

- internal/sema/analyze.go (new file)
  - `Analyze(src string) ([]Diagnostic, error)` — single-file parse+Resolve+Check.
  - `HasErrors(diags []Diagnostic) bool`.
  - `Diagnostic.Render(filename string) string` — no src arg (Line/Col on Pos).
- internal/token/token.go
  - `OffsetToPosition(src string, off int) Pos` — pure offset->Line/Col, clamped;
    new home for the deleted check.OffsetToPosition. Kept import-free (manual loop)
    so token stays a zero-import leaf.

Both build clean (`go build ./internal/sema/ ./internal/token/`). These coexist
with internal/check. Cutover (next step) repoints consumers; cleanup deletes
internal/check and the legacy corpus tests.
