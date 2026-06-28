# Generate Manifest Over Existing Corpus — Business Specification

## Overview

The goal golden corpus currently lives as loose files scattered across feature
directories and testdata folders. This feature indexes those existing files into
a single manifest so any runner (current pipeline, future AST backend, etc.) can
discover and consume every golden case without knowing the on-disk layout. No
source file is moved or rewritten — only an index is produced.

## Functional Requirements

### FR-1: Index transpile pairs
The system SHALL index every goal source paired with an expected-Go golden,
drawn from both `features/NN/examples` and the top-level `testdata` directory, as
a transpile case.

### FR-2: Index checker cases
The system SHALL index every checker source under `testdata/check` as a check
case.

### FR-3: Non-destructive generation
Generation SHALL NOT move, rename, or modify any indexed source file.

### FR-4: Emit a manifest file
The system SHALL write the index to `corpus/manifest.json` at the repository
root, in the established `corpus.Manifest` JSON shape.

### FR-5: Deterministic output
Repeated generation over an unchanged corpus SHALL produce byte-identical
output (stable ordering), so the manifest is diffable.

## Acceptance Criteria

- [ ] The generated manifest contains exactly 51 transpile pairs.
- [ ] The generated manifest contains exactly 50 check cases.
- [ ] No source file under `features/` or `testdata/` is modified by generation.
- [ ] `corpus/manifest.json` exists and loads via `corpus.Load` without error.
- [ ] Regenerating produces identical output.

## User Interactions

Maintainers regenerate the manifest via a generator command (and a
`go:generate` directive). Test code consumes the manifest through the existing
`corpus.Load`.

## Error Handling

If the corpus root cannot be read, generation SHALL return a descriptive error
rather than writing a partial manifest.

## Out of Scope

- Building the runners that consume the manifest (later stories US-003+).
- Indexing inline package-mode tests (US-009).
- Honoring `// want` markers (US-004).

## Open Questions

None — counts and layout are audited and fixed.
