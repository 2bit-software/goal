# Verify: Acceptance Coverage — US-021

Full gates green: `go build ./...`, `go vet ./...`, `go test ./... -count=1` all
pass. `go list -deps ./internal/interp` contains no internal/backend,
internal/typecheck, or go/types (US-022 envelope intact).

| Acceptance criterion | Evidence |
|----------------------|----------|
| Derive evaluated field-by-field using resolved sema types, applying from-registry conversions for bridged fields | `TestDeriveTotalNestedStructAndRegistryBridge`: asserts identity (Name), nested struct recursion (Home Addr->AddrV2), and a registry bridge (Zip string->Code via `from func parseCode`). |
| Unit test over a 12-derive-convert shape asserts a derived conversion produces the expected target struct | Same test — program is the `derive_nested_struct` shape; asserts the full PersonV2 value. |
| Fallible derive returns target on success, propagates error on failure | `TestDeriveFallibleSucceedsAndReturnsNilError` (Typed + nil error) and `TestDeriveFalliblePropagatesConversionError` (non-nil error value "empty id"). |
| Unsourced/unconvertible field yields a descriptive error, not a silent zero | `TestDeriveUnsourcedFieldIsRefused`: asserts the error contains "not sourced" and names field B. |

No CRITICAL/MAJOR findings.

## Assumptions
- `errors.New` (host bridge) is used to manufacture the fallible test's error
  value, since the interpreter has no `&convErr{}` address-of for the fixture's
  original error construction.
