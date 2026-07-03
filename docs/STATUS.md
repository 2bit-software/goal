# goal — current language status (verified 2026-07)

> **Authoritative, empirically-verified snapshot.** The older planning docs
> (`ROADMAP_TO_GOAL.md`, `TODO.md`) are **token-splice-era history** and understate
> how complete the language is. This file is the live picture. Every claim below is
> backed by a `go run ./cmd/goalc <probe>.goal` result or a source `file:line`.
> Scope: the Go-transpile path (`goalc`/`goal`); the `goscript` interpreter is separate.

## Architecture, as built

A real AST front-end: `internal/lexer` → `internal/parser` → `internal/sema` (lexical
checker) → `internal/backend` (lowering to Go). A typed depth stage,
`internal/typecheck`, runs `go/types` over the **lowered** Go for the checks the lexical
stage must defer. (This supersedes the retired "no new parser / lexer-only" plan the old
roadmap still describes — see `REWRITE-ARCHITECTURE.md`.)

## (a) Complete / working

All 12 features (enums, match, Result, Option, `?`, closed-E, implements, no-zero-value,
assert, doctests, derive-convert) lower and check. Notably — cases the stale docs list as
"deferred" that in fact **work today** (each probed, `goalc` exit 0):

| Construct | Probe result |
|-----------|--------------|
| `Result`/`Option` stored as a struct field, function parameter, or slice element (§8.7) | lowers clean |
| closed-E `?` across matching error enums | lowers clean |
| `derive func` over structs resolvable in-file/import | lowers clean |
| bind-then-`match` a stored `Result` (`r := f(); match r {…}`) | lowers clean |

## (b) Deferred to real types — the depth stage cannot close these

These stay honest `Warning`s (`unresolved-match-enum`/`-sealed`, `unresolved-error-enum`,
`unresolved-derive-field`). They are **not** reachable by the current typed depth stage,
for a structural reason worth recording:

- The depth stage loads types with `importer.Default()` (`internal/typecheck/typecheck.goal:106`),
  which resolves stdlib but **cannot import sibling or third-party goal packages**
  (`DECISIONS.md:1743-1749`).
- It runs on **lowered** Go. The only cases the lexical stage defers are *unresolvable
  types* — and an unresolvable enum/struct also **blocks the backend from lowering** the
  `match`/`derive`/`?`, so no Go is produced for the depth stage to inspect.
- Therefore `{cases sema defers} ∩ {cases the depth stage can resolve} = ∅`. The three
  depth checks that *do* exist (`implements`/`must-use`/`no-zero-value`) work only because
  their targets lower fine; these deferrals do not.

Closing them needs the front end to resolve more cross-package facts, or the importer swap
in (d) — not another depth check.

## (c) Deliberate design boundaries (located diagnostics, working as intended)

- `?` is valid **only on an assignment right-hand side** (`x := f()?` / `x = f()?`).
  A multi-value or complex LHS is a located `[question-assign-unsupported]`
  (`internal/guide/catalog.goal:44`); a callee not returning `(value, error)` is
  `[question-binds-nonvalue]` (`catalog.goal:40`).

## (d) Genuinely open — needs a decision, not just engineering

- **Inline `?` in expression position** (`f(g()? + 1)`): parses (produces `ast.UnwrapExpr`)
  but the backend refuses it with an ICE-shaped `backend: unsupported expression
  *ast.UnwrapExpr`. Feature 05 *deferred* (did not refuse) this; lifting it means hoisting
  the early-return out of an arbitrary expression — real, bounded backend work behind a
  design nod.
- **Nested `Result.Err(variant)` match patterns**: rejected at parse; the design says
  compose a nested `match`. Revisit only as a deliberate surface decision.
- **Value-position `x := match` with non-inferable arm types**: rejected
  (`internal/backend/emit.goal:1860`) — asks for `var x T = match …`. Correct without arm-type
  inference; closing it needs inferred types the backend doesn't hold pre-lowering.
- **Importer swap** `importer.Default()` → `importer.ForCompiler(…, "source", …)`
  (`DECISIONS.md` DEPTH-TODO, open): would let the depth stage see sibling packages and
  surface cross-package Go type errors. Cross-cutting; spike before committing.

## What "next" actually looks like

The language is substantially complete for its scope. The remaining work is **design
decisions** (the (d) items) plus one cross-cutting **importer** spike — not the pile of
"missing features" the historical docs imply. Pick from (d) deliberately.
