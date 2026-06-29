# Implementation Plan — US-004

## File Inventory

### New Files
None.

### Modified Files
| File | Changes |
|------|---------|
| `internal/fix/resultsig.go` | Make `fixResultSig` conservative about call-site safety: refuse to convert exported functions and functions referenced anywhere but a collapsible error-propagation call site. Add helpers `safeLocalPropagationCalls` and `unsafeRefOffset`; split per-decl structural analysis into `classifyResultSig` returning a candidate or a skip report. |
| `selfhost/**/*.goal` | Whatever `goal fix --inplace selfhost` produces after the fixer is corrected. Expected: no changes (every conversion candidate is an unsafe cross-boundary conversion). |
| `prd.json` | Set US-004 `passes: true`. |
| `progress.txt` | Append the US-004 entry; add any reusable Codebase Pattern. |

## Dependency Graph

1. Correct `internal/fix/resultsig.go` (the fixer must be safe before it is run).
2. Rebuild `goal`; run `goal fix --inplace selfhost`; verify idempotence.
3. Run gates (`task check`, `task build`, `task fixpoint`) + fix unit tests.
4. Commit; mark `passes:true`; log progress.

## Interface Contracts

```go
// classifyResultSig examines one decl. Returns a candidate when the function has the
// convertible (T, error) shape with all-conforming returns; otherwise (nil, *Report)
// for a near-miss worth surfacing, or (nil, nil).
func classifyResultSig(src string, d ast.Decl, info *sema.Info, decls map[string]string) (cand *sigCand, rep *Report)

type sigCand struct {
    fn          *ast.FuncDecl
    nameLine    int
    successT    string
    res         *ast.FieldList
    successReps []textedit.Replacement
}

// safeLocalPropagationCalls returns the offsets of local-call Fun idents that sit in a
// collapsible error-propagation binding inside a will-be-Result function — the only call
// sites a converted function's 2-tuple use becomes a valid `?`.
func safeLocalPropagationCalls(src string, file *ast.File, info *sema.Info, decls map[string]string, successTOf map[string]string, willResult func(string) bool) map[int]bool

// unsafeRefOffset returns the offset of the first reference to name that is neither the
// declaration nor a safe call site, and whether one exists.
func unsafeRefOffset(file *ast.File, name string, safe map[int]bool) (int, bool)
```

`fixResultSig` flow:
1. Build candidate list + emit the existing near-miss Skips (bare error, multi-value, non-propagating return) via `classifyResultSig`.
2. Compute `successTOf` (candidate name -> success type) and `willResult(name)` (candidate or already Result/Option).
3. Compute the safe call-site offset set.
4. For each candidate: if exported -> Skip ("exported `X` has callers fix cannot see; not auto-converted to Result"); else if it has an unsafe reference -> Skip ("`X` is called where `?` cannot apply ...; not auto-converted to Result"); else emit the signature + success replacements and record the change. The previous exported `Warn`-and-convert branch is removed.

## Integration Points

- `cmd/goal` `fix` command calls `fix.File` unchanged.
- `fix.File`'s fixed-point loop calls `fixResultSig` unchanged; only its internals change.
- `fixPropagate` / `fixPropagateInit` / `reportCallSites` unchanged.

## Testing Strategy

- `go test ./internal/fix` — all existing cases must stay green:
  `TestConvertTupleToResult` (unexported `load` with a collapsible call site in
  `describe` still converts), `TestDecoratedErrorNotConverted`,
  `TestMultiValueNotConverted`, the propagate/option/init-guard/match cases.
- `go test ./cmd/goal` — `TestFixExportedWarning` still sees an "exported"+name
  report on stderr (now a Skip); the unexported `load` stdout/inplace/directory
  tests still convert.
- `goal fix --inplace selfhost` then a second run: no file changes
  (`git status` clean on selfhost, idempotent).
- `task check`, `task build`, `task fixpoint`: green.
