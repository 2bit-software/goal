# Implementation Tasks — US-001 Define cap capability model

## Task 1: Implement internal/cap package
**Status**: completed
**Files**: `internal/cap/cap.go`, `internal/cap/cap_test.go`
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2, FR-3, FR-4, FR-5
**Verify**: `go build ./internal/cap && go vet ./internal/cap && go test ./internal/cap -count=1`

### Instructions
- Create `internal/cap/cap.go`, `package cap`, stdlib only.
  - `type Capability int` with iota constants in order: `Stdout, Stdin, FileRead,
    FileWrite, Net, Concurrency, Time, Env`.
  - `func (c Capability) String() string` — switch returning the readable name;
    default returns `"Capability(N)"`.
  - `func allCapabilities() []Capability` — ordered slice of all eight.
  - `type CapabilitySet struct { bits uint64 }`.
  - `func (s CapabilitySet) Has(c Capability) bool` — `s.bits&(1<<uint(c)) != 0`.
  - `func (s *CapabilitySet) Grant(c Capability)` — `s.bits |= 1 << uint(c)`.
  - `func GrantAll() CapabilitySet` — loop allCapabilities(), Grant each.
  - `func DenyAll() CapabilitySet` — return zero-value CapabilitySet{}.
- Create `internal/cap/cap_test.go`, `package cap`, stdlib `testing` only (NO testify):
  - `TestGrantAllHasEvery`: for each c in allCapabilities(), assert GrantAll().Has(c).
  - `TestDenyAllHasNone`: for each c in allCapabilities(), assert !DenyAll().Has(c).
  - `TestGrantAddsCapability`: s := DenyAll(); s.Grant(Net); assert s.Has(Net) and
    !s.Has(Stdout).
  - `TestStringNonEmpty`: for each c, assert String() != "" and not the unknown form.

## Task 2: Write the restriction-diff doc
**Status**: completed
**Files**: `docs/goscript/restriction-diff.md`
**Depends on**: Task 1
**Spec coverage**: FR-6
**Verify**: file exists and lists all eight capabilities with a default-grant column.

### Instructions
- Create `docs/goscript/restriction-diff.md`.
- Open with the principle (REWRITE-ARCHITECTURE.md §4): capabilities are authority,
  not language difference; goscript restricts by ungranted host authority. v1 grants
  everything by default via GrantAll; deny is host opt-in.
- Table with columns: Capability | What it authorizes | goscript default.
  One row per capability (Stdout, Stdin, FileRead, FileWrite, Net, Concurrency,
  Time, Env), all defaulting to Granted.
- Note that enforcement at effect sites is later runtime work (forward-reference the
  later stories, not this one).
