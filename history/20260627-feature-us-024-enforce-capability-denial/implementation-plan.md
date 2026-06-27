# Implementation Plan — US-024 Enforce capability denial

## File Inventory

### New Files

| File | Purpose |
|------|---------|
| `internal/interp/cap_deny_test.go` | Unit tests: a printing program under denied Stdout raises a located, named capability-denied error and writes nothing; the same program under granted Stdout still prints. |

### Modified Files

| File | Changes |
|------|---------|
| `internal/interp/interp.go` | Add a typed `CapabilityError{Cap cap.Capability, Pos token.Pos}` with a located, named `Error()` string. Change `emitStdout` to accept the effect's source position and, on a denied capability, return that error WITHOUT calling `write`. Grant path unchanged. |
| `internal/interp/host.go` | Pass the `fmt.Println` call's source position (`sel.Pos()`) into `emitStdout`. |

## Package Structure

No new packages. All changes stay in `internal/interp`, reusing the
dependency-free `internal/cap` (already imported) and `internal/token` (already
imported in interp.go).

## Dependency Graph

1. `CapabilityError` type + `emitStdout` signature change (`internal/interp/interp.go`) — depends only on existing `cap` + `token` imports.
2. Call-site update to pass position (`internal/interp/host.go`) — depends on 1.
3. Tests (`internal/interp/cap_deny_test.go`) — depend on 1 and 2.

## Interface Contracts

```go
// CapabilityError is the located, named refusal raised when a host effect is
// attempted without the capability that authorizes it.
type CapabilityError struct {
    Cap cap.Capability
    Pos token.Pos
}

func (e CapabilityError) Error() string
// => "interp: <line:col>: capability denied: <Cap> not granted"

// emitStdout now takes the effect's source position; on denied Stdout it
// returns CapabilityError{Cap: cap.Stdout, Pos: pos} and performs no write.
func (ip *Interp) emitStdout(pos token.Pos, write func(io.Writer) error) error
```

## Integration Points

- `internal/interp/host.go::evalHostCall` is the sole `emitStdout` caller (the
  `fmt.Println` interception). It already holds `sel` (`*ast.SelectorExpr`), so
  it passes `sel.Pos()` as the position argument.
- `token.Pos.String()` provides the located `line:col` rendering (already used
  by gate() and the unresolved-host-symbol refusal).

## Testing Strategy

`internal/interp/cap_deny_test.go`, stdlib `testing` only (no testify), mirrors
the existing `cap_io_test.go` setup (parser.ParseFile + sema.Resolve + New with
options, captured `bytes.Buffer` sink):

- `TestPrintlnUnderDeniedStdoutIsRefused`: run `printlnProgram` with
  `WithCapabilities(cap.DenyAll())` + `WithStdout(&buf)`; assert Run returns a
  `CapabilityError` (errors.As), the error names Stdout, carries a position, and
  `buf` is empty.
- `TestEmitStdoutDeniedReturnsLocatedNamedError`: call `emitStdout` directly with
  a denied set and a sentinel write func; assert the write func never ran (sink
  empty) and the returned error is a located, named `CapabilityError`.
- `TestPrintlnUnderGrantedStdoutStillPrints`: regression — under `GrantAll` the
  program prints and Run returns nil (FR-5).

## Requirement Traceability

- FR-1/FR-4: emitStdout denied branch returns error, skips write -> denial tests.
- FR-2 (named): `CapabilityError.Cap` + message includes capability name.
- FR-3 (located): `CapabilityError.Pos` + `Pos.String()` in message.
- FR-5: granted path unchanged -> regression test.
