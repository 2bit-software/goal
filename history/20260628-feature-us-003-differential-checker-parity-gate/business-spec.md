# Differential Checker Parity Gate (US-003) — Business Specification

## Overview

Before the legacy `internal/check` lexical checker can be deleted (US-005), we
must prove the new AST-based `sema` checker produces equivalent findings over the
committed check corpus. This story adds an automated parity gate: a test that
runs both checkers over every check case and fails if they disagree on any
finding that is not an explicitly documented divergence.

## Functional Requirements

### FR-1: Run both checkers over the whole check corpus
The gate runs both the legacy checker (`check.Analyze`) and the AST checker
(`SemaCheck`) over every `KindCheck` case in the committed manifest (the cases
under `testdata/check/**`). It fails loudly if the manifest yields no check cases.

### FR-2: Compare findings by a stable key
Each finding is projected to the tuple (file, line, feature, code, severity) and
the two checkers' finding multisets are diffed. Message wording is excluded — the
gate is about which guarantee fires where, at what severity.

### FR-3: Identical except for a documented allowlist
The gate passes only when the two checkers' finding sets are identical, except
for divergences listed in an explicit allowlist embedded in the test. Every
allowlist entry must be backed by a note in `DECISIONS.md`. An undocumented
divergence fails the gate; a stale allowlist entry (documented divergence that no
longer reproduces) also fails the gate.

### FR-4: Document the divergences
Each of the four discovered divergences is recorded in `DECISIONS.md`. For the
three cases where the AST checker fires an Error and the legacy checker deferred
(Warning), the divergence is recorded as a documented improvement, and the case's
`// want` markers reflect the sema behavior (already satisfied — the markers
already contain the sema Error message substrings).

## Acceptance Criteria

- [ ] A test runs both the sema checker and the legacy `internal/check` over
      every case under `testdata/check/**` and compares findings by
      (file, line, feature, code, severity).
- [ ] The test passes only when findings are identical, except for divergences
      explicitly recorded in `DECISIONS.md`.
- [ ] Any divergence where the AST checker fires (Error) and the legacy checker
      deferred is recorded in `DECISIONS.md` as a documented improvement, and the
      corresponding `// want` markers reflect the sema behavior.
- [ ] `task check` and `task build` are green.

## Error Handling

- Manifest load failure or a checker internal error → test fatal.
- Undocumented divergence → test failure naming file/line/feature/code/severity
  and which checker produced it.
- Stale allowlist entry → test failure prompting removal from the allowlist and
  DECISIONS.md.

## Out of Scope

- Rewiring `cmd/goal check` onto sema (US-004).
- Deleting `internal/check` (US-005).
- Any change to checker logic.

## Open Questions

- None. The complete divergence set was discovered empirically (4 entries) and is
  documented.
