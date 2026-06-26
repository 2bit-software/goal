# Feature Specification: `goal fix`

**Feature Branch**: `main` (no branch — user requested no branching)
**Created**: 2026-06-25
**Status**: Approved (AutoMode)
**Input**: "Make a `goal fix` command that checks for common Go patterns that can be
translated into goal — e.g. explicit error returns where the previous line could've been
a `?`, which means detecting `(T, error)` signatures that should be `Result[T, error]`.
Fix has two modes: print to stdout, and `-inplace` to write the file back. Audit other
things we can easily detect and fix."

## Overview

`goal fix` is a migration aid. goal is a superset of Go, so a `.goal` file can be written
in plain-Go style and still compile. `fix` finds those plain-Go patterns and rewrites them
into idiomatic goal — the inverse direction of the lowering passes. It modernizes code the
way `gofmt`/`go fix` reshape Go: read source, apply mechanical rewrites, print to stdout by
default or write back with a flag. It is **best-effort** — `goal check` remains the
authority on correctness, and `fix` never claims to have produced compiling code; it
produces *more idiomatic* code and reports anything it could not safely transform.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Collapse manual error/nil propagation to `?` (Priority: P1)

A developer has a `.goal` function that already returns `Result[T, error]` (or
`Option[T]`) but whose body still does the verbose Go dance: `v, err := g(); if err != nil
{ return Result.Err(err) }`. They run `goal fix` and the boilerplate collapses to
`v := g()?`. This is the exact case the request names ("the previous line could've been
just a `?`"), and it is fully body-local: no signature changes, no caller changes.

**Why this priority**: Highest signal-to-risk ratio. Zero blast radius (the function's
contract is unchanged), unambiguous detection, and it is the transform the user explicitly
asked for. Delivers a usable tool on its own — a viable MVP.

**Independent Test**: Feed a `.goal` file containing a `Result`-returning function with a
manual `if err != nil` propagation block; assert stdout shows the same file with that block
collapsed to `v := expr?`, and that re-running `fix` on the output changes nothing.

**Acceptance Scenarios**:

1. **Given** a function returning `Result[T, error]` whose body contains
   `v, err := g(args)` immediately followed by `if err != nil { return Result.Err(err) }`,
   **When** `goal fix` runs, **Then** those two statements become `v := g(args)?` and the
   rest of the file is byte-for-byte unchanged.
2. **Given** the discarded-value shape `_, err := g(); if err != nil { return
   Result.Err(err) }`, **When** `goal fix` runs, **Then** it becomes `_ := g()?`.
3. **Given** an `Option[T]`-returning function with `o := g(); if o == nil { return
   Option.None }` followed by a use of `*o`, **When** `goal fix` runs, **Then** it becomes
   `v := g()?` with `*o` uses rewritten to `v`.
4. **Given** an `if err != nil` block that does **not** bare-propagate (it wraps/decorates:
   `return Result.Err(fmt.Errorf("...: %w", err))`, logs, or returns a different value),
   **When** `goal fix` runs, **Then** it is left untouched (not a `?` candidate).
5. **Given** a file already free of collapsible propagation, **When** `goal fix` runs,
   **Then** output equals input (idempotence) and the file is reported as unchanged.

---

### User Story 2 — Convert `(T, error)` signatures to `Result[T, error]` (Priority: P2)

A developer has a `.goal` helper still written as plain Go: `func load(p string) ([]byte,
error)` with manual propagation throughout and `return data, nil` / `return nil, err` exits.
`goal fix` rewrites the signature to `Result[[]byte, error]`, converts the return
statements (`return v, nil` → `return Result.Ok(v)`; bare `return zero, err` collapses into
the preceding call's `?`; decorated errors → `return Result.Err(e)`), applies the US1
collapse to the body, and updates call sites it can see within the discovered package set.

**Why this priority**: This is the "detect `(T, error)` that should be `Result`" half of the
request. It is higher-value but carries blast radius: a signature change forces every caller
to change. Safe within the package set; exported functions may have unseen external callers.

**Independent Test**: Feed a package with a `(T, error)` helper and one same-package caller;
assert the helper's signature, returns, and body convert, the caller's call site converts to
`?`/`match`, and an exported helper additionally emits a warning that external callers may
break.

**Acceptance Scenarios**:

1. **Given** `func f(...) (T, error)` whose body uses only the bare-propagation idiom and
   ends in `return v, nil`, **When** `goal fix` runs, **Then** the signature becomes
   `Result[T, error]`, `return v, nil` becomes `return Result.Ok(v)`, `return zero, err`
   propagations collapse to `?`, and decorated-error returns become `return Result.Err(e)`.
2. **Given** a same-package caller `x, err := f(); if err != nil { return zero, err }` inside
   another `Result`/`Option` function, **When** `goal fix` runs, **Then** the call site
   becomes `x := f()?`.
3. **Given** a caller that is **not** inside a `Result`/`Option` function (so `?` is illegal
   there), **When** `goal fix` runs, **Then** that call site is left unchanged and reported
   as a manual-follow-up site (rather than producing un-compilable `?`).
4. **Given** an **exported** function `func F(...) (T, error)`, **When** `goal fix` rewrites
   its signature, **Then** a warning is emitted naming the function and noting external
   callers outside the scanned path may need manual updates.
5. **Given** a function returning **more than one** non-error value (`(A, B, error)`) or no
   value (`error` only), **When** `goal fix` runs, **Then** the signature is **not**
   converted (out of MVP mapping) and the function is reported as skipped with the reason.

---

### User Story 3 — Catalog & report additional fixable patterns (Priority: P3)

Beyond the two error transforms, `goal fix` detects other plain-Go→goal opportunities and
either fixes the body-local-safe ones or reports the rest as suggestions, so the developer
gets a single audit of how to make a file more idiomatic.

**Why this priority**: Broadens value and satisfies "audit other things we can easily
detect and fix," but each extra fixer multiplies surface area. Implemented after P1/P2.

**Independent Test**: Feed a file containing a `switch` over a value of an in-file `enum`
type; assert it is rewritten to an exhaustive `match`, and that a `*T`-returning sentinel
function is *reported* (not auto-converted in MVP) as an `Option[T]` candidate.

**Acceptance Scenarios**:

1. **Given** a `switch s { case Enum.A: ...; case Enum.B: ... }` over a value whose type is a
   declared in-file `enum`, **When** `goal fix` runs, **Then** it becomes
   `match s { Enum.A => ...; Enum.B => ... }`, with any missing variants surfaced (so the
   exhaustiveness checker can flag them) — body-local, no contract change.
2. **Given** a function `func find(id ID) *User` that returns a `nil` sentinel, **When**
   `goal fix` runs, **Then** it is **reported** as an `Option[User]` candidate but not
   auto-converted (signature ripple — deferred), unless P2's signature machinery covers it.

### Edge Cases

- **Ambiguous propagation**: `if err != nil` whose body does anything other than `return
  <zero…>, err` / `return Result.Err(err)` → not a candidate; leave untouched.
- **Multiple error names / shadowing**: detection keys on the variable bound on the
  immediately preceding `:=`/`=` line, not a fixed name `err`, so shadowed or renamed error
  variables are handled and unrelated `if x != nil` checks are not misfired.
- **`?` already present**: idempotent — already-collapsed code is not re-processed.
- **File that does not lex / is malformed**: report the file as skipped with a located
  message; never emit partially-mangled output for that file.
- **No `.goal` files under path**: error "no .goal packages found under <path>" (same as
  build/check).
- **`-inplace` with zero changes**: do not rewrite the file (no spurious mtime churn);
  report it as unchanged.
- **Comments and formatting inside an edited region**: a leading comment on a collapsed
  block, or a non-trivial multi-line call expression, must be preserved or the block must be
  treated as not-safely-fixable rather than dropping the comment.
- **Trailing `return v, nil` not at function end / multiple returns**: every `(value, error)`
  return in a converted function is rewritten consistently (`Ok`/`Err`/collapse), not just
  the last one.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The CLI MUST provide a `goal fix [flags] [path]` subcommand, registered in the
  umbrella CLI's command list and help/usage output.
- **FR-002**: `path` MUST accept either a single `.goal` file or a directory; a directory is
  scanned recursively for `.goal` files using the same discovery rules as `build`/`check`
  (skipping hidden/`_`-prefixed dirs and `testdata`). Omitted `path` defaults to `.`.
- **FR-003**: By default (no write flag), `fix` MUST print the rewritten source to stdout
  and MUST NOT modify any file on disk.
- **FR-004**: With `-inplace`, `fix` MUST write the rewritten source back to each changed
  file in place, leave unchanged files untouched, and report which files were written.
- **FR-005**: `fix` MUST collapse the manual error-propagation idiom into `?` inside any
  function that returns `Result[T, error]` or `Option[T]`, covering the keep (`v := …?`),
  discard (`_ := …?`), and Option (`v := …?` with `*o` rewrite) shapes — and MUST NOT
  collapse `if err != nil` blocks that wrap, log, or return non-propagating values. (US1)
- **FR-006**: `fix` MUST convert a function whose signature is exactly `(T, error)` (one
  non-error value) into `Result[T, error]`, rewriting its `(value, error)` return statements
  to `Result.Ok`/`Result.Err`/`?`-collapse, and MUST skip — with a reported reason —
  functions returning multiple non-error values or `error`-only. (US2)
- **FR-007**: When `fix` changes a function's signature, it MUST update call sites within the
  discovered package set where the rewrite is legal (caller is itself `Result`/`Option`-
  returning → `?`), MUST leave illegal call sites unchanged, and MUST report every call site
  it could not rewrite as a manual-follow-up. (US2)
- **FR-008**: When `fix` changes the signature of an **exported** function, it MUST emit a
  warning naming the function and noting that callers outside the scanned path may break.
- **FR-009**: `fix` MUST detect a `switch` over a value of a declared in-file `enum` type
  and surface it as a `match` candidate. (US3, body-local.) **Implementation note**: shipped
  as *detection/report* only — goal `match` arms are single expressions, so a faithful
  mechanical rewrite of statement-bodied Go `switch` clauses is not expressible; the
  auto-rewrite is deferred. The detection delivers the audit value safely.
- **FR-010**: All rewrites MUST be minimal splices — only changed regions are edited;
  untouched code (including formatting and comments outside edited regions) MUST be
  preserved byte-for-byte. `fix` MUST NOT reflow or reformat the whole file.
- **FR-011**: `fix` MUST reach a fixed point — repeated runs MUST converge so that some run
  produces no further changes (a single run need not be terminal: e.g. converting a
  signature to `Result` in run 1 may expose a now-legal `?` collapse in run 2; run 3 is a
  no-op). No fixer may oscillate.
- **FR-012**: When `fix` cannot safely transform a detected candidate (ambiguous, malformed,
  unmapped shape), it MUST leave that code unchanged and report it as a skipped suggestion
  with a located (`file:line`) message, rather than emitting incorrect or partial output.
- **FR-013**: Diagnostics, warnings, and skip/suggestion reports MUST go to stderr; rewritten
  source (default mode) and write confirmations (`-inplace`) MUST go to stdout — matching the
  existing `build`/`check` stream conventions.
- **FR-014**: `fix` MUST exit non-zero only on operational failure (bad path, unreadable
  file, malformed flags); producing suggestions or warnings alone MUST NOT fail the command.

### Detection & Safety Rules (mechanical — from audit)

These pin down "safely fixable" so the implementer does not guess. The governing principle:
**when any condition below is unmet, do not transform — report the candidate as a
suggestion (FR-012).** Never emit a partial or guessed rewrite.

- **DR-1 (bare propagation, the `?` candidate)**: A block is collapsible iff it is exactly
  `if <e> != nil { return <r> }` where (a) `<e>` is the error variable *bound on the
  statement immediately preceding the `if`* (an `:=`/`=` whose LHS includes `<e>`; no other
  statement may sit between that binding and the `if`), and (b) `<r>` is a pure propagation:
  for Result, `<r>` is `<zero…>, <e>` or `Result.Err(<e>)`; for Option, the block is
  `if <o> == nil { return <zero…> | Option.None }`. Any other body (wrapping, logging,
  returning a different value, multiple statements) → not collapsible.
- **DR-2 (zero-value match)**: The non-error operands in `<zero…>` are compared **textually**
  (whitespace-trimmed) against `zeroLit(T, Tables.TypeDecls, …)` (the same generator used by
  `internal/pass/defaults.go`). A return value that is neither the computed zero nor (for the
  terminal success return) the bound value blocks the transform for that function.
- **DR-3 (signature conversion is all-or-nothing)**: `(T, error)` → `Result[T, error]`
  proceeds only if **every** `return _, err` in the function is a DR-1 bare propagation and
  **every** `return v, nil`-shaped exit returns exactly one non-error value. If any return is
  decorated/non-conforming, the signature is left as-is and the function is reported skipped
  with the offending `file:line`. Only single-non-error-value signatures are mapped.
- **DR-4 (Option `*o` rewrite scope)**: Collapsing `o := g(); if o == nil { return … }` to
  `v := g()?` proceeds only if `o` is not referenced before the nil check and every later use
  of `o` in the function is `*o` up to the next rebinding of `o`; those `*o` become `v`. If
  `o` escapes (passed where a `*T` is wanted, captured by a closure, used un-dereferenced),
  the block is not fixable → reported.
- **DR-5 (comments)**: `scan.Lex` discards comments, so a fixer cannot see them in the token
  stream. Before collapsing/removing a multi-statement region, check the **raw source** span
  for `//` or `/*`; if the edited span contains a comment, treat the block as not-safely-
  fixable and report it (never silently drop a comment).
- **DR-6 (multi-line call)**: In the MVP, if the assignment RHS being collapsed spans more
  than one source line, treat it as not-safely-fixable and report it (avoids mangling
  continuation/formatting). Single-line RHS only.
- **DR-7 (call-site legality)**: A call site of a converted function is rewritable to `?` iff
  it occurs directly in the body of a function whose name resolves in
  `Tables.FuncSignatures` to `ModeResult`/`ModeOption`. Calls inside closures, `defer`, or
  `go` statements are not rewritable → reported as manual follow-up.
- **DR-8 (enum switch)**: A `switch <scrut> { … }` is a `match` candidate iff the lexical
  type of `<scrut>` (or its case labels' qualifier) names a key in `Tables.Enums`. Qualified
  cross-package enum types are out of scope.
- **DR-9 (exported warning is conservative)**: `fix` cannot prove external callers exist; the
  FR-008 warning states the signature changed and external callers *may* need manual updates.

### Key Entities

- **Fixer**: an independent rewrite rule with a name (e.g. `propagate`, `result-sig`,
  `match`), a detector, and a splicer. Fixers compose over one file's source + tables.
- **Change**: a single applied rewrite — file, line span, before/after, and the fixer that
  produced it. Drives stdout output and `-inplace` decisions.
- **Suggestion / Skip report**: a detected-but-not-applied opportunity or an unsafe case —
  file, line, message — emitted to stderr.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On a `Result`-returning function with N bare-propagation blocks, `goal fix`
  collapses all N into `?` and changes nothing else in the file (verified byte-for-byte).
- **SC-002**: Running `goal fix` repeatedly reaches a fixed point — for the MVP fixers,
  output stabilizes within two runs (run N+1 over run N's output yields no change).
- **SC-003**: Running `goal check` on `fix`'s output either (a) passes, or (b) fails only on
  call sites `fix` explicitly reported as manual follow-up — `fix` never introduces a silent
  type error in code it claims to have fixed.
- **SC-004**: `fix` never silently produces incorrect code: every candidate it does not fix
  is reported, and no untouched region of any file is altered.
- **SC-005**: Default mode writes nothing to disk (verified: no file mtimes change);
  `-inplace` writes exactly the changed files and no others.

## Testing Requirements *(mandatory)*

### Test Strategy

- **Golden-file unit tests** for the `internal/fix` package: pairs of `before.goal` /
  `after.goal` per fixer (propagation collapse, signature conversion, call-site update,
  switch→match), plus idempotence (`after.goal` fed back yields itself) and "no-change"
  fixtures. stdlib `testing` only (no testify — project is zero-dependency).
- **Round-trip correctness**: for representative `after.goal` outputs, run the existing
  transpile/check pipeline and assert it passes (catches rewrites that produce invalid goal).
- **CLI integration tests** in `cmd/goal/main_test.go` using the `goalModule(t, files)` +
  in-process `run(args, &out, &errOut)` pattern: assert (a) default mode prints rewritten
  source and writes no files, (b) `-inplace` writes exactly the changed files, (c) stderr
  carries warnings/suggestions, (d) exported-signature warning fires, (e) exit codes.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001/002 | Integration | `goal fix` dispatches; file and dir paths both work; `.` default |
| FR-003/005 | Golden + Integration | Default prints collapsed-`?` source, writes nothing |
| FR-004 | Integration | `-inplace` writes changed files only; unchanged left alone |
| FR-005 | Golden | keep / discard / Option propagation shapes collapse; wrapped errors don't |
| FR-006 | Golden | `(T,error)`→`Result`; multi-value / error-only skipped with reason |
| FR-007 | Golden + Integration | same-package call sites updated; illegal sites reported |
| FR-008 | Integration | exported-signature change emits warning |
| FR-009 | Golden | `switch` over in-file enum → `match` |
| FR-010 | Golden | byte-for-byte preservation outside edited spans |
| FR-011 | Golden | idempotence: `after.goal` → no change |
| FR-012 | Golden + Integration | ambiguous/malformed candidates left unchanged + reported |
| FR-013/014 | Integration | stream routing and exit codes |

### Edge Case Coverage

- Wrapped/decorated `if err != nil` → not collapsed (FR-005 negative fixture).
- Shadowed/renamed error variable → detection keys on preceding binding, not name `err`.
- `(A, B, error)` and `error`-only signatures → skipped with reason (FR-006).
- Caller outside a Result/Option function → left unchanged, reported (FR-007).
- Malformed/non-lexing file → skipped with located message, no partial output (FR-012).
- `-inplace` with zero changes → no file rewritten (FR-004/SC-005).

## Out of Scope (MVP)

- Auto-converting `*T`-sentinel functions to `Option[T]` (signature ripple) — **reported**
  as a suggestion only (US3 scenario 2); the conversion machinery lands with P2 but the
  pointer→Option detector stays suggestion-only until proven safe.
- Migrating open `error` returns into **closed** error enums (needs cross-package impact
  analysis and a lint-policy decision; goal-design-spec §3.3).
- Updating call sites in packages **outside** the scanned path, or in non-`.goal` Go files.
- Struct field-completeness and `implements`-clause insertion (these are checker concerns,
  not idiom translations).
- Reformatting / `gofmt`-style normalization of untouched code.

## Open Questions (resolved for MVP)

- *Diff vs full-source stdout?* MVP prints full rewritten source (gofmt-style). A `-d`
  unified-diff and `-l` list-changed-files mode are natural follow-ups, not MVP.
- *Whole-package vs single-function call-site updates?* MVP updates call sites within the
  discovered package set only; everything else is reported. `goal check` is the safety net.
