# Research — US-009 Port sema package to goal

## Summary

This is a mechanical port following the well-established pattern from
US-005 (token), US-006 (lexer), US-007 (ast), US-008 (parser). No external
research needed — the harness (internal/selfhost.BuildTranspiled +
BuildAndTest) and the port_test convention already exist in-repo.

## Findings

1. **Port = verbatim copy.** internal/sema/*.go (non-test) are valid goal
   (Go superset). Reserved-word scan for bare `match`/`enum`/`assert`
   identifiers found only string-literal occurrences (e.g. "10-assert",
   "assert-always-true") — zero identifier collisions, so files copy verbatim.

2. **Dependencies.** sema imports in-module token, ast, parser, and
   pass-through foreign go/parser, go/format, go/types (foreign.go). The
   deps-aware BuildTranspiled layout + BuildAndTest deps map handle in-module
   deps; foreign stdlib imports pass through unchanged.

3. **Behavioral gate test-file selection.** Mirroring US-007/US-008, only
   self-contained white-box test files belong in the throwaway temp module:
   no repo-relative ../../features fixtures, no testdata/extpkg directory
   dependency. foreign_test.go reads internal/sema/testdata/extpkg, so it is
   excluded. The remaining suites must be checked for self-containment before
   inclusion.

## Confidence

High — fourth port in an identical, proven sequence.

## Open questions

- Exactly which sema test files are self-contained (no testdata, no shared
  cross-file helpers). Resolved during implementation by inspecting each
  *_test.go and running the gate.
