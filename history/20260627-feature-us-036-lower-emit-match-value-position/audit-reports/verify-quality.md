# Verification — Quality (US-036)

## Code quality
- The enum-match lowering mirrors the known-good legacy `internal/pass/match.go`
  encoding but reads the parsed arms + `sema.Enum.FieldSet` instead of scanning
  tokens, consistent with the US-033/034/035 approach.
- Reuses existing machinery (`matchQualifier`, `enumOf`, `usesIdent`, `gensym`,
  `renames`); the only new emitter state is `armBinding`/`armFields`, saved and
  restored per arm so nesting is clean.
- `selectorExpr` field export is guarded on the active arm binding + the variant
  field set, so ordinary selectors (`io.Writer`, `c.n`) are untouched.
- `tryVarMatch` claims only a single-name, single-value, explicitly-typed `var`
  whose value is an enum match; everything else falls through to the ordinary
  decl emitter.
- Format-once discipline preserved: emit writes token-correct Go, GoFormatter
  normalizes layout.

## Tests
- Behavioral tier (build+vet) over all 5 cases + an encoding test pinning the
  lowering shapes. No testify; stdlib `testing` only.

## Minor / follow-ups
- Gensym `v` vs the splice engine's `__goal_v`: intentional (US-035 retired the
  magic prefix); exact goldens regenerate in US-042.
- Statement/value `match` arm bodies route through the existing `armBody`, so a
  block-bodied arm in value position would be ill-formed — not produced by the
  parser for value positions and not present in the corpus.

## Assumptions
- No new package needed; all changes in `internal/backend` + corpus fixtures +
  one count-test line.
