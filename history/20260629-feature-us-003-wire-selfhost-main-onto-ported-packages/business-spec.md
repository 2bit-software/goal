# US-003 Wire self-host main onto ported packages — Business Specification

## Overview

The goal-written compiler must be built from the ported goal packages rather
than the trusted Go `internal/*` packages, so it is genuinely self-contained.
This converts the byte-identical bootstrap fixpoint from a trivial check into a
real differential self-host proof — the oracle the later idiomatic audits rely
on. This is the verbatim stage: no idiomatic rewrites.

## Functional Requirements

### FR-1: Self-contained selfhost imports
The selfhost compiler entrypoint and every package it transitively depends on
SHALL import the `selfhost/*` packages, never any `internal/*` package.

### FR-2: Bootstrap from the self-contained tree
The bootstrap SHALL build `goal-c-1` and `goal-c-2` from the emitted
self-contained selfhost tree (not from the repo's `internal/*` packages).

### FR-3: Genuine byte-identical fixpoint
`goal-c-1` and `goal-c-2` SHALL emit byte-identical Go for the compiler's own
source.

### FR-4: Corpus + behavioral coverage preserved
The goal-built compiler SHALL pass the corpus transpile + behavioral tiers, and
the existing project gates (`task check`, `task build`) SHALL remain green.

## Acceptance Criteria

- [ ] `selfhost/main.goal` imports `selfhost/{token,lexer,ast,parser,sema,project,pipeline,backend}` and no `internal/*` package.
- [ ] No `selfhost/*.goal` file imports any `goal/internal/*` package.
- [ ] `task fixpoint` exits 0 (FIXPOINT OK) with goal-c-1/goal-c-2 built from the self-contained tree.
- [ ] `task check` is green (including the self-host port gates and corpus tiers).
- [ ] `task build` is green.

## User Interactions

CLI / build only: `task bootstrap`, `task fixpoint`, `task check`, `task build`.

## Error Handling

A non-self-contained tree (a leftover `internal/*` import, or an emit dir that is
not a buildable module) surfaces as a `go build` failure during bootstrap or a
non-empty `diff -r` during fixpoint.

## Out of Scope

- Any idiomatic rewrite (enum, match, sealed interface, Result/?) — those are
  US-004 onward.
- Changing the trusted `internal/*` packages or their tests.

## Open Questions

- None. The corpus transpile + behavioral tiers run under `task check` against
  the trusted compiler; the genuine fixpoint proves goal-c reproduces that
  compiler byte-for-byte, so the two together establish the criterion without new
  corpus-through-goal-c infrastructure.
