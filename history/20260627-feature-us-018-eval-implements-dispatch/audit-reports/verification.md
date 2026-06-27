# Verification — US-018 Eval implements dispatch

## Test suite

Full project gates (prd.json verifyCommands) all green:
- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./... -count=1` — all packages ok (internal/interp included)

New test: `go test ./internal/interp -run TestImplements -count=1` — 4/4 pass.

## Acceptance criteria mapping

| Criterion | Evidence | Result |
|-----------|----------|--------|
| Struct satisfying an interface dispatched through it to the correct concrete method at runtime | `TestImplementsValueReceiverDifferentTypes`, `TestImplementsHeterogeneousSliceDispatch` | PASS |
| Unit test over a 07-implements shape calls an interface method on differently-typed values; each concrete impl runs | `TestImplementsValueReceiverDifferentTypes` (Point->"point", Label->"label"); slice test (Square+Rect -> 17) | PASS |
| Value + pointer receiver both dispatch through the interface | value cases + `TestImplementsPointerReceiverThroughInterface` (Counter.Reset mutation observed) | PASS |
| Loud, not silent, on a missing method | `TestImplementsMissingMethodIsLoud` | PASS |
| Verify gates green | build/vet/test all ok | PASS |

## Notes

No production code changed: interface dispatch rides entirely on the US-010
method registry. The address-of operator `&` remains unimplemented and out of
scope; pointer-receiver dispatch is proven without it.
