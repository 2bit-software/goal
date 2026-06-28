# Implementation Plan — US-022 Gate interp on native sema only

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/gate_test.go` | Unit test: non-exhaustive-match program is refused with a located diagnostic; clean program runs; warning-only program is not blocked. Plus the dependency test scanning `go list -deps ./internal/interp` for go/types and goal/internal/typecheck. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/interp/interp.go` | Add a sema-gate at the TOP of `Run()`: run `sema.Check(ip.file, ip.info)`, and if any `sema.Error` diagnostic exists, return a located, named error BEFORE finding/calling main. Add a `gate()` helper + a `RefusalError` type (or formatted error) carrying Pos/Code/Message. |

## Package Structure

```
internal/interp/
  interp.go        (modified — Run() gate)
  gate_test.go     (new — acceptance + dependency tests)
```

No new packages. `internal/sema` is already imported by interp.go.

## Dependency Graph

1. `Run()` gate in interp.go (uses existing `sema.Check`).
2. `gate_test.go` (depends on 1).

## Interface Contracts

```go
// In interp.go — at the top of Run(), before findMain:
func (ip *Interp) Run() error {
    if err := ip.gate(); err != nil {
        return err
    }
    // ... existing findMain + callFunc ...
}

// gate runs the native sema checks and returns a located refusal for the
// first Error-severity diagnostic, or nil when none.
func (ip *Interp) gate() error

// Refusal carries the located diagnostic. Rendered as:
//   "interp: refused before run: <line:col>: [<code>] <message>"
```

The gate iterates `sema.Check(ip.file, ip.info)`, returns on the first
`d.Severity == sema.Error`. Warnings are skipped. A nil file/info yields no
diagnostics (sema.Check is nil-safe over an empty file) so the gate is a no-op
for the trivial program.

## Integration Points

- `internal/interp/interp.go` `Run()` — single integration site; the gate is
  the first statement.
- `internal/sema` `Check` / `Diagnostic` / `Error` — consumed unchanged.

## Testing Strategy

`internal/interp/gate_test.go` (package `interp`, stdlib `testing`, no testify):

- `TestRunRefusesNonExhaustiveMatch`: parse + resolve a program whose `main`
  contains a non-exhaustive `match` on an in-file enum; assert `Run()` returns
  an error whose message contains the diagnostic code (`non-exhaustive-match`)
  and a `line:col` location; assert main's observable effect did NOT happen
  (e.g. it would mutate nothing / the error is returned before eval).
- `TestRunAllowsExhaustiveMatch` (or a clean program): assert `Run()` returns
  nil (no false refusal).
- `TestRunDoesNotBlockOnWarning`: a match over an enum NOT declared in the file
  yields an unresolved-enum WARNING; assert `Run()` is not refused by the gate
  (it may still fail later for an unrelated reason, so assert specifically that
  the returned error, if any, is not the gate refusal).
- `TestInterpHasNoGoTypesOrTypecheckDep`: run `go list -deps ./internal/interp`,
  assert output contains neither `go/types` nor `goal/internal/typecheck`.

Follow neighboring interp tests (interp_test.go) for the parse+resolve+New
harness.
