# Rewire `goal check` onto sema — Business Specification

## Overview

`goal check` runs two stages: a lexical-equivalent stage and a typed depth
stage. Today the lexical stage uses the legacy `internal/check` checker. This
change rewires that stage onto the AST/sema checker so the legacy lexical
checker leaves the live path, without altering the user-visible output of
`goal check`.

## Functional Requirements

### FR-1: AST checker drives the lexical-equivalent stage
`goal check` SHALL produce its lexical-equivalent findings from the AST/sema
checker rather than the legacy lexical checker.

### FR-2: Output unchanged over the corpus
For the corpus check cases, `goal check` output (file, line, severity, code,
message per finding) SHALL be unchanged from before the rewire, except for the
AST/legacy divergences already documented in DECISIONS.md (US-003).

### FR-3: Depth stage and dedup preserved
The typed depth stage SHALL still run, and a depth finding SHALL still suppress
the lexical-equivalent finding for the same construct, identified by file
basename, line, and feature.

### FR-4: Exit and note behavior preserved
A clean package SHALL print `ok` and exit zero; a package with at least one
error-severity finding SHALL exit non-zero. A depth-stage transpile failure
SHALL remain a concise, non-fatal note (no generated-Go dump).

## Acceptance Criteria

- [ ] `goal check` over a corpus check case renders the same findings as before.
- [ ] A depth-backed finding suppresses the lexical-equivalent misfire for the
      same (basename, line, feature).
- [ ] A clean program prints `ok`; a violating program exits non-zero.
- [ ] A depth-stage failure prints "depth stage unavailable" without the
      "--- generated ---" dump.
- [ ] The lexical-equivalent stage no longer routes through the legacy checker.

## User Interactions

CLI: `goal check [path]` — output is `file:line:col: severity: [code] message`
lines on stderr, `ok` on stdout when clean.

## Error Handling

Parse errors are fatal to the check of that package. Foreign-import resolution
failures are non-fatal (types left deferred). Depth-stage transpile failures are
a non-fatal note.

## Out of Scope

- Deleting the `internal/check` package (that is US-005).
- Migrating `internal/typecheck` off `internal/check`'s `Severity` type or off
  analyze/scan (US-007).
- Any change to the diagnostics the checkers emit beyond the documented
  divergences.

## Open Questions

None.
