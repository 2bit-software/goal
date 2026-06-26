# Business Spec: Goal Language Server — Diagnostics (Layer 2, Milestone 1)

**Initiative**: 6a3dd9a5-feature-lsp-server
**Status**: Spec complete
**Created**: 2026-06-25

## Problem / Motivation

The VS Code extension (Layer 1) colors `.goal` files but cannot tell the developer
when their code is *wrong*. Goal already rejects invalid programs with precise,
located errors (non-exhaustive match, must-use Result, no-zero-value, etc.), but
those errors are only visible by running `goal check` in a terminal. Developers
have to leave the editor, run a command, read offsets, and map them back to source.

Layer 2 closes that loop: the same guarantees that `goal check` enforces should
appear as inline red/yellow squiggles in the editor as you type, with hover text
explaining the violated guarantee.

## Scope of this milestone

**Diagnostics only.** This is the first, independently shippable slice of Layer 2.
Hover, go-to-definition, completion, rename, and semantic-token highlighting are
explicitly deferred to later milestones.

## Functional Requirements

- **FR-001**: When a developer opens or edits a `.goal` file in VS Code, the editor
  displays the language's check violations inline as diagnostics (squiggles), without
  the developer running any command.
- **FR-002**: Each diagnostic is anchored to the correct location in the **original
  `.goal` source** (the code the developer sees), not generated Go.
- **FR-003**: Diagnostics carry a severity that matches Goal's own classification:
  guarantee violations show as **Errors**, advisory deferrals show as **Warnings**.
- **FR-004**: Each diagnostic shows the human-facing message Goal already produces,
  attributed to source "goal", and includes Goal's stable error code (e.g.
  `non-exhaustive-match`) and feature name (e.g. `02-match`).
- **FR-005**: Diagnostics reflect the **current unsaved buffer**, not the last saved
  file — editing updates them; they do not require a save.
- **FR-006**: When a file becomes valid, its squiggles clear. When a file is closed,
  its diagnostics are removed.
- **FR-007**: All violations in a file are reported together (not just the first), so
  the developer sees the full set at once.
- **FR-008**: The editor integration must not require the developer to hand-configure
  paths in the common case — installing the extension and having the `goal` tool
  available should be enough.

## Acceptance Criteria

- Opening `editors/vscode/examples/sample.goal` (made invalid, e.g. a non-exhaustive
  `match`) shows a red squiggle on the offending construct within ~1s of opening.
- The squiggle's range covers the construct at its real line/column in the `.goal`
  file; hovering shows Goal's message and code.
- Removing the offending code clears the squiggle without saving.
- A file with multiple distinct violations shows one diagnostic per violation.
- A valid `.goal` file shows zero diagnostics.
- Severity: an `Error`-class guarantee renders as an error; a deferral renders as a
  warning.
- The server emits nothing to stdout except framed protocol messages (no corruption);
  diagnostics appear reliably across edits.

## Out of Scope (this milestone)

- Hover, go-to-definition, completion, rename, formatting, code actions.
- Semantic-token (type-aware) highlighting — that is a later Layer 2 increment.
- **Depth / type-backed checks** (`internal/typecheck`: implements, must-use,
  no-zero-value) that require building a `project.Package` and running `go/types`.
  Milestone 1 surfaces only the **lexical checks** (`internal/check`) that run on a
  single in-memory buffer. Type-backed diagnostics are milestone 2.
- Multi-file / cross-file package analysis. Milestone 1 analyzes the open buffer on
  its own (single-file check surface).
- Pull-model diagnostics (`textDocument/diagnostic`); milestone 1 uses push.
- Workspace-wide diagnostics, watching files on disk.

## Open Questions

- **Binary distribution**: ship the server as a `goal lsp` subcommand of the existing
  `goal` binary (recommended — zero new artifacts) vs. a separate `goal-lsp` binary.
  *Decision (default): `goal lsp` subcommand.*
- **Server discovery**: resolve the `goal` binary from `PATH`, from a `goal.lsp.path`
  setting, or bundle it. *Decision (default): a `goal.lsp.path` setting that defaults
  to `goal` on PATH.*
- Single-file lexical checks only in M1 means some cross-file-dependent guarantees
  won't fire until M2. Acceptable for the first slice.

## Success Criteria

- **SC-001**: A developer never has to run `goal check` manually to see guarantee
  violations for the file they're editing.
- **SC-002**: Diagnostics location accuracy: every reported diagnostic points at the
  correct `.goal` line/column (verified against `goal check` CLI output for the same
  file).
- **SC-003**: The server adds zero new Go dependencies (stdlib only), preserving the
  project's zero-dependency posture.
