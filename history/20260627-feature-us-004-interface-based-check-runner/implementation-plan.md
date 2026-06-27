# Implementation Plan — US-004 interface-based check runner

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/corpus/check_runner.go` | Defines the Checker interface + CheckerFunc adapter and RunCheck(root, Case, Checker) which parses inline `// want` markers and matches diagnostics by line. |
| `internal/corpus/check_runner_test.go` | TestCheckRunner: loads the manifest, runs every KindCheck case through CheckerFunc(check.Analyze), fails loudly on zero cases. |

### Modified Files
None.

## Package Structure
```
internal/corpus/
  corpus.go            (Case/Manifest/Load — existing)
  runner.go            (Transpiler runner — existing, US-003)
  check_runner.go      (NEW — Checker runner)
  runner_test.go       (existing; manifestPath const lives here)
  check_runner_test.go (NEW)
```

## Dependency Graph
1. internal/check (existing) — provides Diagnostic, Severity, Error, OffsetToPosition, Analyze.
2. internal/corpus check_runner.go — depends on internal/check + internal/corpus Case.
3. check_runner_test.go — depends on 2 and the committed manifest.

No import cycle: internal/check imports only internal/analyze; corpus may import check.

## Interface Contracts
```go
// Checker analyzes goal source and returns located diagnostics.
type Checker interface {
    Check(src string) ([]check.Diagnostic, error)
}

// CheckerFunc adapts a free func (e.g. check.Analyze) to Checker.
type CheckerFunc func(src string) ([]check.Diagnostic, error)
func (f CheckerFunc) Check(src string) ([]check.Diagnostic, error) { return f(src) }

// RunCheck executes one KindCheck Case against ck.
func RunCheck(root string, c Case, ck Checker) error
```

Marker semantics (mirrors internal/check/check_test.go):
- regexp: `//\s*want\s+"([^"]*)"`, per-line list of expected substrings.
- Each marker satisfied by a diagnostic on the same line whose Message contains substr.
- Any Error-severity diagnostic on a line with no markers => unexpected rejection => fail.
- Warnings may go unclaimed.

## Integration Points
- RunCheck mirrors RunTranspile in runner.go (same root-join, same case-id error style).
- Test mirrors TestTranspileRunner in runner_test.go; reuses manifestPath + repoRoot.

## Testing Strategy
- TestCheckRunner iterates m.Cases, skips non-KindCheck, runs RunCheck(repoRoot, c, CheckerFunc(check.Analyze)) in a t.Run subtest per case ID; t.Fatalf if zero ran.
- verifyCommands gate: go build/vet/test ./....
