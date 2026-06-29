# Implementation Plan — US-008 idiomatic audit: parser

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| (none) | The audit outcome is a recorded DECISION with no `.goal` source change. |

### Modified Files
| File | Changes |
|------|---------|
| `DECISIONS.md` | Append a "self-host idiomatic audit — US-008 (parser)" section recording the three classified candidates and their refusals-with-reason, mirroring the US-005/006/007 sections. |
| `prd.json` | Set US-008 `passes: true` (done after green verify, per loop-runner). |
| `progress.txt` | Append the US-008 iteration entry (per loop-runner). |

## Package Structure
No change to `selfhost/parser/` (parser.goal, goal_construct.goal, goal_decl.goal,
goal_match.goal, goal_stmt.goal stay byte-identical). Only the documentation ledger
and loop bookkeeping files change.

## Dependency Graph
1. Confirm machine check + verify gates green on the unchanged source (already done
   in research; re-confirmed at verify step).
2. Append the DECISIONS.md US-008 section (depends on 1's findings).
3. Set prd.json `passes:true` and append progress.txt (depends on 2 + green verify).

## Interface Contracts
Unchanged. The public contract pinned by the oracle stays:
```
func ParseFile(src string) (*ast.File, error)
```
The internal error-accumulator contract stays: `parser.errs []error` +
`(*parser).errorf(pos token.Pos, format string, args ...any)`; internal helpers
return bare AST nodes / `nil`.

## Integration Points
- `internal/selfhost` port gate (`BuildTranspiled` compile gate + `BuildAndTest`
  behavioral gate against `selfhost/../parser/parser_test.go`) runs under
  `task check` — the "tests pass against the transpiled package" gate.
- `task fixpoint` — whole-tree byte-identical goal-c-1/goal-c-2 — the oracle.
- `goal fix selfhost/parser/*.goal` — the AC-2 machine check.

## Testing Strategy
No new tests. Reuse the existing port gate + fixpoint. Verification is:
1. `goal fix selfhost/parser/*.goal` → no content diff, no auto-convertible sites.
2. `task check` (includes the selfhost port gate + `internal/parser` tests).
3. `task build`.
4. `task fixpoint` → FIXPOINT OK, byte-identical.

## Why no source change (trace to spec)
- FR-1 (Result/?): parser is an error-accumulator; no intra-package `(T,error)`
  propagation surface; the lone `(T,error)` is the oracle-pinned exported
  `ParseFile`. Refusal recorded.
- FR-2 (match): no in-file `enum` declared; all switches are over non-enum
  scrutinees. Refusal recorded.
- FR-3 (public API preserved): satisfied trivially (no change).
