# Technical Requirements / Research — US-001

From REWRITE-ARCHITECTURE.md §10 Phase 0.1:

- Manifest is a generated JSON index over the *existing* corpus files (do not
  move them). Per case: `{id, kind: transpile|check|doctest, input, expected,
  mode: file|package, normalize: gofmt}`.
- Zero-dependency constraint: use stdlib `encoding/json` only. Tests use stdlib
  `testing` (no testify).
- This story only defines the `Case`/`Manifest` model and `Load`; later stories
  (US-002+) generate the actual manifest and build runners.

## Design

- `Kind`, `Mode`, `Normalize` as string-typed enums with exported constants so
  JSON round-trips human-readably.
- `Manifest` is a struct holding `[]Case` (plus optional metadata) so it can grow.
- `Load(path string) (Manifest, error)` reads + JSON-unmarshals the file.
