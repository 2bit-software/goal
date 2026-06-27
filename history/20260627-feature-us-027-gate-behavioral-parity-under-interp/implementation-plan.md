# Implementation Plan — US-027

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/corpus/interp_gate_test.go` | The whole-corpus interpreter behavioral gate (`TestInterpWholeCorpusBehavioralGate`), the package-level `interpGateSkips` skip list, the `blankSkipReasons` helper, and the focused `TestInterpGateSkipListRejectsBlankReason` unit test. |

### Modified Files
| File | Changes |
|------|---------|
| (none) | No production code changes — the feature composes existing seams (`corpus.RunInterp`, `corpus.Load`, `manifestPath`, `repoRoot`). |

## Package Structure

```
internal/corpus/
  interp_runner.go            (existing) RunInterp
  interp_runner_test.go       (existing) TestInterpRunner
  ast_gate_test.go            (existing) parallel Go-engine gate — precedent
  interp_gate_test.go         (NEW)      interpreter whole-corpus gate
```

All in `package corpus` (the existing internal tests already live there;
internal/corpus imports internal/interp with no cycle).

## Dependency Graph

1. `interpGateSkips` (empty map literal) + `blankSkipReasons` helper — no deps.
2. `TestInterpWholeCorpusBehavioralGate` — depends on 1, plus existing
   `Load`/`manifestPath`/`repoRoot`/`RunInterp`/`KindDoctest`.
3. `TestInterpGateSkipListRejectsBlankReason` — depends on 1.

## Interface Contracts

```go
// case ID -> justification; empty today.
var interpGateSkips = map[string]string{}

// blankSkipReasons returns the sorted IDs whose reason is blank (whitespace-only
// counts as blank).
func blankSkipReasons(skips map[string]string) []string

func TestInterpWholeCorpusBehavioralGate(t *testing.T)
func TestInterpGateSkipListRejectsBlankReason(t *testing.T)
```

## Integration Points

- `corpus.Load(manifestPath)` loads the committed manifest.
- Iterate `m.Cases`, filter `c.Kind == KindDoctest`, skip IDs present in
  `interpGateSkips` (logging the reason), run the rest through
  `RunInterp(repoRoot, c)`.
- Validate the skip list with `blankSkipReasons` (FR-4) and a stale-entry check
  against the manifest's doctest IDs (FR-5).

## Testing Strategy

- `TestInterpWholeCorpusBehavioralGate`: drives every doctest case, fatals on
  empty manifest / zero applicable cases, errors per behavioral failure or
  unjustified/stale skip. This IS the deliverable gate.
- `TestInterpGateSkipListRejectsBlankReason`: feeds `blankSkipReasons` a map with
  good/blank/whitespace reasons and asserts exactly the blank IDs come back,
  proving FR-4's enforcement fires even while the real skip list is empty.
- stdlib `testing` + `reflect.DeepEqual` only; no testify.
