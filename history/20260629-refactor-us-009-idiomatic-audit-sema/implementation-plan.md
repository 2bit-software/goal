# Implementation Plan — US-009 Idiomatic audit: sema

## File Inventory

### New Files
None.

### Modified Files
| File | Changes |
|------|---------|
| `selfhost/sema/analyze.goal` | Convert `Analyze` to return `Result[[]Diagnostic, error]`, using `parser.ParseFile(src)?` for propagation and `return Result.Ok(Check(file, info))` for the success path. Update the doc comment to say it returns a `Result` whose error is a parse failure. |
| `DECISIONS.md` | Add a "self-host idiomatic audit — US-009 (sema)" section recording the one conversion (Analyze) and every refusal-with-reason. |
| `prd.json` | Set US-009 `passes: true`. |
| `progress.txt` | Append the US-009 entry (and any reusable pattern to the Codebase Patterns block). |

## Package Structure

No structural change. selfhost/sema/*.goal stays a 12-file goal package. Only
analyze.goal's `Analyze` body/return type changes.

## Dependency Graph

1. Edit `selfhost/sema/analyze.goal` (Analyze -> Result/?).
2. Verify: `goal fix` no-diff, sema port gate, `task check`, `task build`,
   `task fixpoint` (depends on 1).
3. Record DECISIONS.md, flip prd.json, append progress.txt (depends on 2 green).

## Interface Contracts

Before:
```
func Analyze(src string) ([]Diagnostic, error) {
	file, err := parser.ParseFile(src)
	if err != nil {
		return nil, err
	}
	info := Resolve(file)
	return Check(file, info), nil
}
```

After:
```
func Analyze(src string) Result[[]Diagnostic, error] {
	file := parser.ParseFile(src)?
	info := Resolve(file)
	return Result.Ok(Check(file, info))
}
```

ModeResult (open-E `Result[T, error]`) lowers to native `([]Diagnostic, error)`,
so the emitted Go signature and body are behaviorally identical. No other
signatures change.

## Integration Points

- `Analyze` has zero consumers in the selfhost tree and zero oracle tests
  (verified via grep), so no caller or test edits are required.
- `parser.ParseFile(src)` returns `(*ast.File, error)` (ModeResult) — a valid
  open-E `?` source inside an `error`-returning Result function (feature-05).

## Testing Strategy

No new tests (single-package audit; the conversion preserves emitted behavior and
the gate has no Analyze test by design — adding one is out of scope). Verification
is by the project gates:
- `goal fix selfhost/sema/*.goal` (per file) → no source diff; only documented
  skip/suggestion reports remain.
- `go test ./internal/selfhost -run TestPortedSemaPackage` (behavioral gate).
- `task check`, `task build`, `task fixpoint` (FIXPOINT OK, byte-identical).

## Refusals (to be recorded in DECISIONS.md)

EnrichForeign (accumulator []error), foreignDecls (multi-value + accumulator
caller), DefaultResolver (DirResolver-type + oracle-pinned), goListResolve
(tail-return at pinned boundary), AnalyzePackageInDir/With (exported + oracle-pinned
+ multi-value), constIntLit/moduleResolve/readModulePath (comma-ok control-flow
bool), Mode/Severity (ordered exported iota ints, cross-package == and conversions),
no switch->match (no in-file enum; Diagnostic.Code/Feature are strings).
