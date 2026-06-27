# Reify Package-Mode Tests As Fixtures — Business Specification

## Overview

The golden corpus is the runner-independent yardstick every goal front-end is
judged against. Today the only multi-file *package*-mode test sources live
inline inside Go test files (`internal/pipeline/foreign_test.go` and
`pipeline_package_test.go`), invisible to the corpus manifest. This feature
lifts those inline sources into on-disk multi-file package fixtures, declares
the imports each fixture needs, indexes them as `Mode=package` corpus cases, and
adds a corpus runner that executes them — so package-mode conformance is part of
the same shared suite as single-file cases.

## Functional Requirements

### FR-1: On-disk multi-file package fixtures
The two inline package sources SHALL exist as on-disk fixtures, each a directory
of one or more `.goal` files:
- a cross-file package (an enum/Result split across two files sharing one prelude), and
- a foreign-derive package (a `derive func` over a struct imported from a foreign Go package).

### FR-2: Declared import map
Each package fixture SHALL declare its import map: the import paths it references
and, for each, where the foreign Go package lives. A fixture with no imports
declares an empty map.

### FR-3: Manifest indexing as Mode=package
The corpus manifest SHALL index each package fixture as a `Mode=package` case,
without disturbing the existing single-file case counts (51 transpile / 50
check / 4 doctest).

### FR-4: Package-mode runner
The corpus SHALL provide a runner that, given a package-mode case, transpiles the
package through a pluggable seam and verifies it passes.

## Acceptance Criteria

- [ ] The cross-file and foreign-derive package sources exist as on-disk
      multi-file fixtures (no longer only inline in Go test files).
- [ ] Each package fixture declares an import map.
- [ ] The generated corpus manifest contains the package fixtures as
      `Mode=package` cases.
- [ ] The existing single-file counts (51 transpile, 50 check, 4 doctest) are
      unchanged.
- [ ] The corpus runner executes every `Mode=package` case and all pass.
- [ ] `go build ./...`, `go vet ./...`, and `go test ./... -count=1` are green.

## User Interactions

Developer-facing only. Maintainers regenerate the manifest with the existing
generator (`go run ./cmd/corpus-gen -root .`) and run `go test ./...`; the
package-mode cases run automatically as part of the corpus suite.

## Error Handling

- A package fixture that fails to transpile, emits invalid Go, or fails to
  compile SHALL fail its case with a descriptive, case-identified error.
- A runner invoked with zero package-mode cases SHALL fail loudly (guard against
  a silently empty suite), consistent with the other corpus runners.

## Out of Scope

- Adding new language behaviors; this only relocates and indexes existing cases.
- Behavioral tiers for single-file cases (already delivered in US-007/008).
- Removing the inline Go test sources is optional cleanup, not required by this
  story (the fixtures are the durable artifact).

## Open Questions

- None blocking. The import map's foreign packages resolve in-module via the
  existing `DefaultResolver`; no external module access is required.
