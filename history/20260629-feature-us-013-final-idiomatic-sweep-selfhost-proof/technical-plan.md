# Technical Plan — US-013 Final idiomatic sweep and self-host proof

## Strategy

This is a proof + documentation story. Reconnaissance confirms the selfhost tree is
already at the `goal fix` fixed point (a whole-tree `goal fix -inplace` on a copy yields
an empty diff). No `.goal` source change is required or wanted — changing source would
risk the byte-identical fixpoint oracle. The only artifact change is DECISIONS.md, which
must gain a US-013 section that (a) records the whole-tree proof and (b) documents the
one file never given a per-package audit: `selfhost/main.goal` (`run`, `emitPackage`).

## Changes

### C-1: DECISIONS.md — add US-013 section (FR-1, FR-2)
Append a new section "self-host idiomatic audit — US-013 (final whole-tree sweep +
self-host proof)" after the US-012 section. Contents:
- The whole-tree machine proof: `goal fix -inplace` across all 39 `selfhost/**/*.goal`
  produces no diff; stderr is only `skipped`/`suggestion`, no `fixed`.
- A roll-up that every flagged function maps to a documented refusal (US-005..US-012 by
  package) and names the one previously-undocumented file.
- The `selfhost/main.goal` refusals:
  - `run (error)` — bare-error CLI entry point (called by `main`); propagation wraps
    usage messages via `fmt.Errorf` or is top-level plumbing; `goal fix` `skipped:
    [result-sig]` ("returns a bare `error`"). Bare-error has no value channel; Result
    buys nothing and the rule refuses it. Mirrors typecheck `Load`/`Check` (US-012).
  - `emitPackage (error)` — bare-error IO helper called only by `run`; propagates
    os.MkdirAll/os.WriteFile errors. Same refusal class.
- Self-host proof statement: `task fixpoint` byte-identical + corpus tiers green.

No interface/signature changes. No source files touched.

## Dependency Ordering

1. C-1 (DECISIONS.md edit).
2. Verification gates (no code dependency between them; run in prd order).

## Verification

Per prd.json verifyCommands + AC checks:
1. `goal fix -inplace` over a copy of selfhost -> empty `diff -r` vs original; stderr
   has no `fixed` lines. (FR-1, AC-1/AC-2-of-story)
2. `task check` — vet + full test suite incl. internal/corpus tiers and
   internal/selfhost behavioral port gates. (FR-4)
3. `task build`.
4. `task fixpoint` -> `FIXPOINT OK`. (FR-3)

## Files

- `DECISIONS.md` (append US-013 section) — the only repository source/doc change.
- `prd.json` (US-013 passes:true) — bookkeeping at finalize.
- `progress.txt` (append entry) — bookkeeping at finalize.
