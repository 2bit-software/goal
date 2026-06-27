# Plan Audit: Buildability — US-001

- Dependency order is valid: corpus.go (stdlib only) → fixture json → test. No
  forward references.
- Interface contracts are concrete Go signatures; `Load` returns `(Manifest, error)`.
- File paths verified: `internal/corpus/` does not yet exist (new package); no
  conflict with existing `internal/{analyze,check,pipeline,...}`.
- Each component compiles independently: corpus.go has no intra-repo deps.
- No integration points to wire in this story.

No CRITICAL/MAJOR findings.

## Assumptions
- JSON is the manifest encoding (stdlib encoding/json), per Phase 0.1.
- `Manifest` is a struct wrapping `[]Case` (not a bare slice) for future metadata.
