# Technical Requirements / Research — US-002

## Existing model (US-001)

`internal/corpus` already defines `Case{ID, Kind, Input, Expected, Mode,
Normalize}`, the `Manifest` struct, and `Load(path)`. US-002 adds generation
over the real corpus on top of this model.

## Corpus layout (audited)

- `features/*/examples/*.goal` with sibling `*.go.expected` → 40 transpile pairs
  (includes feature 11-doctests examples, which are counted as transpile pairs).
- `testdata/*.goal` (top-level, non-recursive) with sibling `*.go.expected` →
  11 transpile pairs.
- `testdata/check/**/*.goal` → 50 check cases. Checker expectations live inline
  as `// want "substr"` markers in the source (parsed by the US-004 runner), so
  the generated check Case carries an empty Expected and Normalize=none.

## Approach

- Add `Generate(root string) (Manifest, error)` to `internal/corpus` that walks
  the three locations relative to `root`, builds a deterministically-ordered
  `[]Case`, and returns the Manifest. No source files are moved.
- Add a small generator command (`cmd/corpus-gen`) plus a `//go:generate`
  directive that writes `corpus/manifest.json`.
- The test computes the repo root from the package dir, calls `Generate`, and
  asserts the transpile/check counts.

## Constraints

- Zero-dependency: stdlib `testing` only, no testify.
- Paths stored in the manifest are repo-root-relative and slash-separated for
  portability.
