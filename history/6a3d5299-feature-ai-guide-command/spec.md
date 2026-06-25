# Feature Specification: `goal ai` — binary-sourced AI bootstrap guide

**Feature Branch**: `feat/ai-guide-command`
**Created**: 2026-06-25
**Status**: Draft (decisions confirmed with user)
**Input**: "We have an AI-KNOWLEDGE-BOOTSTRAP.md file that bootstraps an AI on using this
language. Pivot to a `--ai` flag/command on `goal` that outputs how to use the language
directly from the binary. Wherever it makes sense the output should be generated from the
actual capabilities of the language/transpiler, so changing the language immediately
changes the output. This may require extra per-feature docs that get sourced for this help."

## Summary

Replace the hand-maintained `AI-KNOWLEDGE-BOOTSTRAP.md` with a `goal ai` subcommand
(aliased `goal --ai`) that emits a complete, self-contained "how to write goal" guide to
stdout as Markdown. The guide is assembled at invocation time so it cannot drift from the
real toolchain:

- **Feature examples + their lowered Go** are re-transpiled live through the real
  `pipeline.Transpile` at output time — the lowering shown is exactly what *this* binary
  produces, with no intermediate regeneration step.
- **The CLI/toolchain section** (commands, flags, iteration loop) is rendered from the
  binary's own command registry.
- **The diagnostics catalog** (the checker's stable error codes) is rendered from the live
  set of codes the checker can emit, and the "what feedback looks like" sample is produced
  by running the checker on an embedded bad snippet.
- **Authored prose** (why each feature exists, locked conventions, authoring do/don'ts) is
  embedded Markdown shipped in the binary, with structural claims guarded by tests.

`AI-KNOWLEDGE-BOOTSTRAP.md` becomes a generated golden artifact: produced by
`goal ai > AI-KNOWLEDGE-BOOTSTRAP.md` and verified by a test, so the in-repo file and the
binary output can never diverge.

## Confirmed Decisions

These were decided with the user up front and are not open questions:

1. **Liveness** — examples are **re-transpiled at output time** (not embedded-and-golden-tested).
2. **Content source** — **reuse `docs/by-example.md`** (the existing live-verified per-feature
   source) for examples/lowerings; **derive CLI commands/flags and diagnostic codes from
   actual code**; new authored sections live as embedded Markdown.
3. **Surface** — a **`goal ai` subcommand** emitting the full guide to stdout; **`goal --ai`**
   accepted as an alias. An optional `goal ai <section>` may emit a single section.
4. **Old file** — `AI-KNOWLEDGE-BOOTSTRAP.md` becomes a **generated golden artifact** guarded
   by a drift test.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Agent bootstraps from the binary (Priority: P1)

An AI coding agent (or a human) lands in a project that uses `goal`, has the `goal` binary
on PATH, and knows nothing about the language. It runs `goal ai` and receives the full,
current, correct guide — surface syntax for every feature, the exact Go each feature lowers
to *as produced by this binary*, the toolchain commands to build/run/check/test, the
checker's diagnostic catalog, and the authoring rules — enough to write and iterate on a
working program without reading any other file.

**Why this priority**: This is the entire point of the pivot. Delivers value on its own even
if nothing else (golden artifact, per-section access) ships.

**Independent Test**: Run `goal ai` on a clean checkout; confirm the output is a single
self-contained Markdown document covering all features, that every shown lowering matches a
live `goal`/`goalc` transpile of the shown source, and that following the "starter program"
section verbatim produces a program that `goal run` executes.

**Acceptance Scenarios**:

1. **Given** the built `goal` binary, **When** `goal ai` is run, **Then** it prints a
   Markdown guide to stdout and exits 0, writing nothing to the working tree.
2. **Given** a feature whose example appears in the guide, **When** that example's `.goal`
   source is transpiled with `goalc`, **Then** the Go it produces is byte-identical to the
   lowering the guide showed for that feature.
3. **Given** the transpiler's behavior for a feature is changed, **When** `goal ai` is run
   again, **Then** the guide's shown lowering for that feature reflects the new behavior
   with no separate doc-regeneration step.

### User Story 2 — Toolchain & diagnostics stay truthful (Priority: P1)

The guide's toolchain section lists exactly the subcommands and flags the binary actually
supports, and the diagnostics section lists exactly the error codes the checker can emit.
Adding, removing, or renaming a subcommand/flag/diagnostic code changes the guide
automatically (or fails a test that forces the catalog to be updated in the same change).

**Why this priority**: The bootstrap file's headline failure mode today is staleness (it
literally warns that README/docs falsely say "checker not started"). Deriving these from
code is what kills that class of drift.

**Independent Test**: Add a throwaway subcommand and a throwaway diagnostic code; confirm the
guide lists them (or that the guarding test fails until the catalog is updated). Remove them;
confirm they disappear.

**Acceptance Scenarios**:

1. **Given** the binary's command registry, **When** the toolchain section renders, **Then**
   every user-facing subcommand and its flags appear with a one-line description, and no
   command absent from the registry appears.
2. **Given** the checker's set of emittable diagnostic codes, **When** the diagnostics
   section renders, **Then** the catalog lists exactly that set (a test fails if the
   authored catalog and the live code set differ).
3. **Given** the embedded bad snippet, **When** the "what feedback looks like" sample
   renders, **Then** it shows the actual diagnostics the checker emits for that snippet,
   in the real `file:line:col: severity: [code] message` format.

### User Story 3 — Repo file never diverges from the binary (Priority: P2)

`AI-KNOWLEDGE-BOOTSTRAP.md` remains in the repo as a browsable artifact (for GitHub, humans,
search) but is generated from `goal ai` and guarded by a test, so it can never silently
drift from what the binary emits.

**Why this priority**: Preserves the convenience of a committed file while removing the
hand-maintenance burden that motivated the pivot. Not required for the command itself to be
useful.

**Independent Test**: Run the generator; confirm the committed file equals `goal ai` output.
Hand-edit the committed file; confirm the golden test fails.

**Acceptance Scenarios**:

1. **Given** `goal ai` output, **When** the golden test runs, **Then** it passes iff the
   committed `AI-KNOWLEDGE-BOOTSTRAP.md` is byte-identical to the output.
2. **Given** a stale committed file, **When** the golden test runs, **Then** it fails with a
   message telling the developer to regenerate.

### User Story 4 — Targeted section access (Priority: P3)

A caller can request a single section (e.g. `goal ai features`, `goal ai toolchain`) to get
just that portion, for cheaper context use.

**Why this priority**: Convenience/optimization; the full dump already satisfies the core
need.

**Acceptance Scenarios**:

1. **Given** a valid section name, **When** `goal ai <section>` is run, **Then** only that
   section is printed.
2. **Given** an unknown section name, **When** `goal ai <section>` is run, **Then** the
   command errors and lists the valid section names.

## Functional Requirements

- **FR-1**: `goal ai` SHALL print a single self-contained Markdown guide to stdout and exit 0
  without modifying the working tree.
- **FR-2**: `goal --ai` SHALL be accepted as an alias producing identical output to `goal ai`.
- **FR-3**: The features portion SHALL be sourced from `docs/by-example.md` (embedded in the
  binary), reusing a single shared parser also used by `cmd/build-playground`.
- **FR-4**: For each feature example, the guide SHALL show the Go produced by running the live
  transpiler on that example's source at invocation time — not a pre-stored output string.
- **FR-5**: A feature whose example is an intentional compile error SHALL show the actual
  located error message the checker/transpiler emits for it.
- **FR-6**: The toolchain section SHALL render the binary's subcommands and flags from a
  command registry that is the same source the CLI dispatch reads, so the two cannot diverge.
- **FR-7**: The diagnostics section SHALL render the checker's stable error codes from the
  live set the checker can emit; a test SHALL fail if the authored catalog omits or invents a
  code relative to that set.
- **FR-8**: The "what the feedback looks like" sample SHALL be produced by running the checker
  on an embedded snippet and printing its real diagnostics.
- **FR-9**: Authored prose sections (intro/why, locked conventions, authoring do/don'ts,
  iteration loop) SHALL be embedded Markdown shipped in the binary; the guide SHALL NOT read
  any file from disk at runtime (it must work from an installed binary anywhere).
- **FR-10**: The feature count and ordering in the guide SHALL be derived from the parsed
  source, not hardcoded.
- **FR-11**: `AI-KNOWLEDGE-BOOTSTRAP.md` SHALL be generatable from `goal ai` output and a test
  SHALL assert the committed file equals that output.
- **FR-12** *(optional, P3)*: `goal ai <section>` SHALL emit a single named section, and an
  unknown section name SHALL error and list valid names.
- **FR-13**: The project's zero-dependency, stdlib-only constraint SHALL be preserved — no new
  third-party dependencies, tests use stdlib `testing` only.

## Acceptance Criteria (testable, consolidated)

- [ ] `goal ai` and `goal --ai` both exit 0, print Markdown to stdout, write nothing to disk.
- [ ] Every feature lowering shown equals a live `goalc` transpile of the shown source
      (verified by a test that transpiles each guide example).
- [ ] The toolchain section lists exactly the registry's subcommands/flags (no more, no less).
- [ ] The diagnostics catalog equals the live code set; the guard test fails on any mismatch.
- [ ] The feedback sample shows real checker output for the embedded snippet.
- [ ] Changing a transpiler behavior changes `goal ai` output with no other regeneration step.
- [ ] Adding a subcommand/flag/diagnostic code surfaces in the guide or fails its guard test.
- [ ] `AI-KNOWLEDGE-BOOTSTRAP.md` golden test passes on a fresh generate, fails on a stale file.
- [ ] No new third-party dependency is introduced; all tests are stdlib `testing`.
- [ ] The shared doc parser is used by both `cmd/build-playground` and `goal ai` (one parser,
      not two).

## Out of Scope

- Changing any language feature, transpiler behavior, or checker diagnostic.
- Authoring deep new per-feature documentation trees (the existing `docs/by-example.md`,
  `features/NN/*/SYNTAX.md`, `TRANSPILE.md`, `goal-design-spec.md`, and `DECISIONS.md` remain
  the deep references; the guide points to them but does not duplicate them).
- Reconciling the stale README/`docs/overview.md` "checker not started" prose (separate
  cleanup; the guide simply renders the truth instead).
- A central refactor that declares every diagnostic code at a single literal site (heavier
  alternative; the chosen approach is an authored catalog guarded by a set-equality test).
- Non-Markdown output formats (JSON/HTML) for `goal ai`.
- WASM/playground UI changes (the playground keeps consuming `features.json` as today).

## Open Questions

- **OQ-1**: Should the optional per-section access (FR-12 / Story 4) ship in v1 or be deferred?
  Recommendation: include section access as a thin layer since the assembly is already
  section-structured, but it is the lowest priority and can be cut without affecting US1–US3.
- **OQ-2**: Where do the new authored sections physically live — a `docs/ai/` directory of
  embedded fragments, or new specially-marked sections appended to `docs/by-example.md`?
  Recommendation: a small `docs/ai/` set of fragments (keeps by-example.md focused on
  examples; gives the guide an explicit assembly order). Resolve during planning.
- **OQ-3**: The checker's depth (typed) stage needs a package/module; the live feedback sample
  will therefore use the lexical stage only (as `goalc` does single-file). Confirm that a
  lexical-stage sample is representative enough, or embed a tiny module fixture to exercise
  the depth stage. Recommendation: lexical sample for v1; note the depth stage in prose.
