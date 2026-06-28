# Implementation Plan — US-003 differential parity gate

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/corpus/parity_test.go` | The differential parity gate test plus the documented-divergence allowlist. |

### Modified Files
| File | Changes |
|------|---------|
| `DECISIONS.md` | Append a US-003 section documenting the four sema↔legacy divergences (three derive-convert improvements + one extra deferral). |

(No `// want` marker edits: the three derive-convert markers already contain the
sema Error message substrings — verified in research.)

## Package Structure

All new code lives in `internal/corpus`, alongside the existing
`check_runner.go` / `sema_checker.go` and their tests. No new package.

## Dependency Graph

1. `DECISIONS.md` divergence section (documentation the allowlist cites).
2. `internal/corpus/parity_test.go` (the gate + allowlist, referencing the doc).

## Interface Contracts

```go
// finding is the comparable projection of a diagnostic.
type finding struct {
    File     string
    Line     int
    Feature  string
    Code     string
    Severity string // "error" | "warning"
}

// divergence is one finding produced by exactly one checker.
type divergence struct {
    finding
    Side string // "sema" | "legacy"
}

// knownDivergences is the DECISIONS.md-backed allowlist (4 entries).
var knownDivergences = []divergence{ ... }

func TestSemaLegacyParity(t *testing.T) { ... }
```

## Integration Points

- Reuses `Load(manifestPath)`, `repoRoot`, `KindCheck` from the corpus package.
- Runs `check.Analyze(src)` (legacy) and `SemaCheck(src)` (AST).
- Projects via `check.OffsetToPosition(src, d.Pos).Line` and `d.Severity.String()`.

## Testing Strategy

- The new test IS the gate. It fails loudly on zero check cases, on any
  undocumented divergence, and on any stale (no-longer-reproducing) allowlist
  entry.
- Regression safety net: existing `TestCheckRunner` (legacy) and
  `TestSemaExhaustiveRunner` (sema) continue to pass unchanged.
- Project gates: `task check`, `task build`.
