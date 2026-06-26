# Initiative: tree-sitter-grammar

**Type**: feature
**Status**: in_progress
**Created**: 2026-06-25
**ID**: 6a3de58b-feature-tree-sitter-grammar

## Steps

| Step | Profile | Status | Updated |
|------|---------|--------|--------|
| spec | feature | complete | 2026-06-25 |
| plan | plan | complete | 2026-06-25 |
| tasks | tasks | complete | 2026-06-25 |
| implement | implement | complete | 2026-06-25 |

## Description

Layer 3: a from-scratch tree-sitter grammar for `.goal` plus highlight queries, for
multi-editor reach (Neovim/Helix/Zed/Emacs) and GitHub rendering — the consumers the
TextMate grammar (L1, VS Code only) and the LSP (L2) don't reach. Built on `main` per
"don't branch"; full feature workflow, AutoMode; CLI via Homebrew `tree-sitter-cli`.

## Goals

- Parse the full repo `.goal` corpus (103 files) with zero ERROR nodes.
- Highlight queries mapping goal's keywords/types/variants/operators/doctests to
  standard tree-sitter captures (portable across hosts).
- Build/verify with the standard tree-sitter toolchain.

## Progress

- **spec**: business-spec, research-summary, technical-requirements-research written.
  Approach fixed: from-scratch focused grammar in `editors/tree-sitter-goal/`, validated
  empirically against the 103-file corpus. tree-sitter-cli 0.26.9 installed via brew.
- Branch note: the initiative tool again auto-created `feat/tree-sitter-grammar` and, this
  time, swept pending checker WIP into two commits (`5c2427c`, `30089e5`) on `main` before
  branching; honored "don't branch" (reverted to `main`, deleted the branch). Those two
  commits + the rest of the WIP were then committed as focused commits during cleanup.
