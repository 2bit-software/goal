# Implementation Plan — US-001 Define cap capability model

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/cap/cap.go` | Capability enum + CapabilitySet (Has/Grant) + GrantAll()/DenyAll() + allCapabilities() |
| `internal/cap/cap_test.go` | Stdlib test: GrantAll().Has(c) true & DenyAll().Has(c) false for every Capability; Grant adds; String sanity |
| `docs/goscript/restriction-diff.md` | Enumerate each capability and goscript's default grant |

### Modified Files
None. Greenfield package + new doc; no existing code depends on this yet.

## Design

### Capability (cap.go)
- `type Capability int` with iota constants in this order: `Stdout, Stdin, FileRead,
  FileWrite, Net, Concurrency, Time, Env`.
- `func (c Capability) String() string` for readable test failures and the doc seam.
- `func allCapabilities() []Capability` returns the ordered slice of all eight.
  Package-private (lowercase) — the test is in-package so it can use it; nothing
  external needs to iterate yet.

### CapabilitySet (cap.go)
- `type CapabilitySet struct { bits uint64 }` — value-copyable bitset; bit `1<<uint(c)`.
- `func (s CapabilitySet) Has(c Capability) bool` — value receiver, total.
- `func (s *CapabilitySet) Grant(c Capability)` — pointer receiver, sets the bit
  (the additive primitive).
- `func GrantAll() CapabilitySet` — set every bit for allCapabilities().
- `func DenyAll() CapabilitySet` — zero value (no bits).

### Doc (docs/goscript/restriction-diff.md)
- Header explaining: capabilities are authority, not language difference
  (REWRITE-ARCHITECTURE.md §4). v1 grants all by default; deny is host opt-in.
- A table: Capability | What it authorizes | goscript default → Granted (all).

## Test Plan (cap_test.go, package cap, stdlib testing only)
- `TestGrantAllHasEvery` — for each c in allCapabilities(), GrantAll().Has(c) is true.
- `TestDenyAllHasNone` — for each c in allCapabilities(), DenyAll().Has(c) is false.
- `TestGrantAddsCapability` — start DenyAll, Grant(Net), assert Has(Net) true and a
  different capability still false.
- `TestStringNonEmpty` — every capability's String() is non-empty and not the
  fallback unknown form (guards the String switch staying in sync with the enum).

## Verification
- `go build ./...`
- `go vet ./...`
- `go test ./... -count=1` (and specifically `go test ./internal/cap -count=1`)
- Confirm `docs/goscript/restriction-diff.md` exists and lists all eight capabilities.

## Conventions
- Zero third-party deps; stdlib `testing` only (no testify).
- New package under `internal/` (house pattern).
