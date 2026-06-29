# Implementation Plan

## File Inventory

### New Files
None.

### Modified Files
| File | Changes |
|------|---------|
| `DECISIONS.md` | Append a new section recording the deliberate decision to keep `selfhost/token`'s `Kind` as an iota-based const block rather than a goal `enum`, with rationale (sealed-interface enum encoding vs array-indexed/range-marker integer Kind) and the related notes (no switch → no match; `Lookup` stays comma-ok; goal fix reports none). |

### Unchanged (deliberately)
| File | Why |
|------|-----|
| `selfhost/token/token.goal` | Audit conclusion: no idiomatic change fits. `Kind` must stay an ordered integer (array index into `kindNames`, `*_beg`/`*_end` range predicates). No `switch` to convert. `Lookup (Kind, bool)` is comma-ok, pinned by the oracle test, and not a goal-fix propagation site. The package is import-free and has no `(T,error)` functions. |

## Package Structure

No structural change. selfhost/token remains a single `token.goal`.

## Dependency Graph

1. Append the DECISIONS.md entry (no code dependencies).
2. Run verification gates.

## Interface Contracts

No interface change. The token public API is preserved exactly:

```
type Kind int
func (k Kind) String() string
func (k Kind) IsLiteral() bool
func (k Kind) IsOperator() bool
func (k Kind) IsKeyword() bool
func Lookup(name string) (Kind, bool)
func IsKeyword(name string) bool
type Pos struct { Offset, Line, Col int }
func (p Pos) Less(q Pos) bool
func (p Pos) IsValid() bool
func (p Pos) String() string
func OffsetToPosition(src string, off int) Pos
type Token struct { Kind Kind; Lit string; Pos Pos }
```

## Integration Points

DECISIONS.md is documentation only — no code integration. The selfhost/token
package is consumed unchanged by selfhost/lexer and the rest of the self-host
tree; preserving its API shape keeps those consumers and the oracle tests
compiling.

## Testing Strategy

No new tests. Verification is by the project gates:
- `goal fix selfhost/token/token.goal` → no diff, no report (FR-2).
- `task check` → runs go vet + full `go test ./...`, including the self-host
  port gate that transpiles selfhost/token and runs internal/token's tests
  against it (FR-3, token tests).
- `task build` → compiler builds.
- `task fixpoint` → goal-c-1 and goal-c-2 emit byte-identical Go (FR-3,
  fixpoint).
