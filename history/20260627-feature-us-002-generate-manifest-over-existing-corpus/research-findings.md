# Research Findings — US-002

This is an internal corpus-indexing task; no external/library research required.
Findings are from a direct audit of the repository.

## Audited corpus counts (verified by find)

- `features/*/examples/*.goal` paired with `*.go.expected`: **40** transpile pairs.
- `testdata/*.goal` (top-level, non-recursive) paired with `*.go.expected`: **11**
  transpile pairs.
- Total transpile pairs: **51**.
- `testdata/check/**/*.goal`: **50** check cases.

These match the prd US-002 notes exactly.

## Existing seam

`internal/corpus` (US-001) supplies the `Case`/`Manifest` types, the `Kind`,
`Mode`, `Normalize` constants, and `Load(path)`. US-002 only needs a `Generate`
function plus a tiny command to emit `corpus/manifest.json`.

## Decisions

- Walk relative to a `root` argument so the generator is testable without `cwd`
  assumptions. The test derives root from the package directory (`../..`).
- Store repo-root-relative, slash-separated paths for portability.
- Check cases use inline `// want` markers, so `Expected` is left empty and
  `Normalize=none`; the dedicated check runner (US-004) parses markers later.
- Deterministic ordering (sorted by path) so the generated manifest is stable
  across runs and diffable.

## Confidence

High — counts verified directly against the working tree; model already exists.
