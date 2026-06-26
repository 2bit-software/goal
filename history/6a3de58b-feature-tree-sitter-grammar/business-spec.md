# Business Spec: tree-sitter grammar for goal (Layer 3)

**Initiative**: 6a3de58b-feature-tree-sitter-grammar
**Status**: Spec complete
**Created**: 2026-06-25

## Problem / Motivation

The TextMate grammar (Layer 1) only colors goal in VS Code, and the LSP (Layer 2)
only reaches editors that run the server. Neither helps Neovim, Helix, Zed, Emacs,
or **GitHub's web view** — all of which highlight via tree-sitter / linguist.

A tree-sitter grammar is the one artifact those consumers share: a structural parser
plus highlight queries that any tree-sitter host can load, with no language server.
It is the multi-editor / GitHub-rendering piece of the editor story.

## Scope of this milestone

A working tree-sitter grammar for `.goal` plus highlight queries, covering goal's
real surface: Go's commonly-used constructs **as they appear in goal code** plus
goal's additions (enum, match, sealed, implements, derive, from, Result/Option, the
`?` unwrap, the `=>` match arrow, `...defaults`/`...derive`, `///` doctests).

## Functional Requirements

- **FR-001**: The grammar parses every `.goal` file in the repository corpus (103
  files) with **no ERROR nodes** — it understands the language as actually written.
- **FR-002**: Highlight queries (`queries/highlights.scm`) assign goal's keywords,
  types, sum types (`Result`/`Option`), enum variants, operators (`=>`, `?`, spreads),
  comments, strings, and numbers to standard tree-sitter highlight captures, so any
  tree-sitter host colors goal sensibly.
- **FR-003**: goal's distinctive constructs are first-class nodes in the tree (not
  generic identifiers): `enum`, `match` arms, `sealed`/`implements`, `derive`/`from`,
  `Result`/`Option` types and constructors, postfix `?`, and `///` doc/doctest comments.
- **FR-004**: The grammar carries no semicolons in source (goal inherits Go's
  automatic semicolon insertion); newline-terminated statements parse without explicit
  separators.
- **FR-005**: The package builds with the standard tree-sitter toolchain
  (`tree-sitter generate` + a C compiler) and is loadable by tree-sitter hosts.

## Acceptance Criteria

- `tree-sitter generate` succeeds with no unresolved grammar conflicts.
- `tree-sitter parse` over all 103 corpus files reports zero ERROR/MISSING nodes.
- `tree-sitter test` (corpus tests) passes for goal-specific constructs (enum decl,
  match with `=>`, `Result`/`Option` types & constructors, `?`, spreads, `///` doctest).
- `tree-sitter highlight` on a sample `.goal` produces highlights for the goal-specific
  tokens (verified by the presence of the expected captures).

## Out of Scope (this milestone)

- 100% fidelity to arbitrary/exotic Go syntax not present in goal code (e.g. rarely
  used Go forms). The target is goal's real surface; the boundary is documented.
- Shipping a compiled `.wasm` and wiring it into the VS Code tree-sitter API (VS Code's
  tree-sitter host API is still stabilizing). The grammar + queries are the deliverable;
  per-editor packaging (Neovim/nvim-treesitter, Zed, wasm) is a follow-up.
- `locals.scm` / `injections.scm` / `folds.scm` beyond highlights (future).
- Publishing to npm / crates.io.

## Open Questions

- **Location**: `editors/tree-sitter-goal/` (sits beside `editors/vscode/`). *Default: yes.*
- **From-scratch vs fork tree-sitter-go**: the corpus is a constrained subset, the repo
  has no LICENSE, and a fork drags in a 2k-line grammar + C scanner. *Decision: from
  scratch, validated against the full corpus* (consistent with the Layer 1 decision).

## Success Criteria

- **SC-001**: A `.goal` file renders with meaningful syntax highlighting in any
  tree-sitter host, with no language server running.
- **SC-002**: The grammar understands 100% of the repo's `.goal` corpus (no ERROR nodes).
