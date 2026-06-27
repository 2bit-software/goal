# Plan Audit: Coverage — US-001

Every acceptance criterion traces to a plan element:

- Case fields (ID/Kind/Input/Expected/Mode/Normalize) → `Case` struct in corpus.go.
- Kind = transpile|check|doctest → `Kind` constants.
- Mode = file|package → `Mode` constants.
- Load valid manifest → `Load` + `TestLoadFixture`.
- Load missing/malformed → error path + `TestLoadMissingFile`/`TestLoadMalformed`.
- Fixture with one case of each Kind, assert every field → `manifest.json` +
  `TestLoadFixture`.

No scope creep: no files beyond the package, its test, and its fixture.
No CRITICAL/MAJOR findings.

## Assumptions
- `Normalize` includes `none` and `gofmt`; only those two are needed now.
