# Technical Requirements / Research — US-001

## From the story notes

- Implements REWRITE-ARCHITECTURE.md §4 / §6 open spec item: the goscript
  restriction diff is "capabilities are authority, not language difference".
- v1 may grant everything via GrantAll, but the seam must exist now.

## Concrete shape (from acceptance criteria)

- New package `internal/cap`.
- A `Capability` enumeration covering at least: Stdout, Stdin, FileRead,
  FileWrite, Net, Concurrency, Time, Env.
- A `CapabilitySet` with:
  - `Has(c Capability) bool`
  - `Grant(c Capability)` (or returns a new set — implementation choice)
  - `GrantAll() CapabilitySet` constructor (holds every capability)
  - `DenyAll() CapabilitySet` constructor (holds none)
- Unit test (stdlib `testing`, NO testify — project is zero-dependency) asserting
  `GrantAll().Has(c)` true and `DenyAll().Has(c)` false for every defined
  Capability. This requires a way to iterate every Capability (e.g. an
  `allCapabilities()` slice).
- `docs/goscript/restriction-diff.md` enumerating each capability and whether
  goscript grants it by default.

## Repo conventions (from progress.txt Codebase Patterns)

- New packages live under `internal/`.
- Tests use stdlib `testing` only — no testify.
- Verify gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
