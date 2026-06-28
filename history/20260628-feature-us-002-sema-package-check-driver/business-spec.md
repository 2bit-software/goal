# US-002 Add sema package-check driver — Business Specification

## Overview

The AST-based semantic checker (`internal/sema`) currently exposes only a
single-file entry point (`Resolve` + `Check`). To replace the legacy lexical
checker on the live path, it needs a package-level driver that checks a whole
multi-file goal package at once, producing per-file diagnostics from merged,
foreign-enriched facts — mirroring the legacy `check.AnalyzePackageInDir` so the
later rewire (US-004) is a near drop-in swap.

## Functional Requirements

### FR-1: Package driver entry point
A package driver SHALL accept the package's source files (as strings) and the
package directory, parse each file, build merged semantic facts across all files,
enrich them with imported (foreign) package facts, and run every sema check over
each file.

### FR-2: Per-file, input-order results
The driver SHALL return one diagnostic list per input file, in the same order as
the input files.

### FR-3: Cross-file and foreign resolution
A diagnostic that can only be produced when facts from a sibling file AND an
imported package are both resolved SHALL be produced by the driver, proving the
merge and the foreign enrichment are both wired in.

### FR-4: Non-fatal foreign-resolution errors
Foreign-import resolution failures SHALL NOT fail the driver; an unresolved
import simply leaves its types deferred (matching the legacy behavior). A
resolver injection point SHALL exist so tests can supply a fake resolver without
the Go toolchain.

## Acceptance Criteria

- [ ] A sema package driver parses each file, calls ResolvePackage, runs
      EnrichForeign, then runs sema.Check per file.
- [ ] The driver returns one `[]Diagnostic` per input file in input order.
- [ ] A test over a multi-file package fixture returns the expected diagnostics,
      including at least one finding that depends on foreign enrichment from
      US-001.
- [ ] `task check` and `task build` pass.

## User Interactions

Internal Go API only (no CLI/UI change in this story). The driver is consumed by
cmd/goal in a later story (US-004).

## Error Handling

- A parse error in any input file is returned as the driver's error result.
- Foreign-import resolution errors are collected and surfaced separately
  (non-fatal); the convenience entry point discards them.

## Out of Scope

- Rewiring cmd/goal onto the driver (US-004).
- Deleting internal/check (US-005).
- Any change to the individual sema checks or their messages.

## Open Questions

None — the shape is dictated by the legacy `check.AnalyzePackageInDir` it mirrors.
