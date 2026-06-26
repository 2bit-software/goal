# Feature Specification: Cross-file & cross-package completeness in the LSP

**Initiative**: `6a3ebcd5-feature-cross-file-completeness`
**Created**: 2026-06-26
**Status**: Draft
**Input**: User description: "while in vscode, i'm seeing errors like `cannot verify 'derive func' is total: target type 'Spec' is not a struct declared in this file — completeness deferred`. Since a large portion of the code reaches across files, this error type is seen a lot, and makes most of the linting useless. Can we update the lsp to handle cross-file checking? it should also handle checking/relationships into non-goal (go only) references."

## Problem

The `goal` language server (`goal lsp`, consumed by the VSCode extension) analyzes each
open file **in isolation**. It calls `check.Analyze(text)`, which builds tables from the
single open buffer (`analyze.Build`) and knows nothing about sibling `.goal` files in the
same package or about imported Go packages.

Because real code constantly references types declared in other files (and in plain-Go
imports), the completeness checks — feature 12 (`derive func` totality), feature 08
(field-completeness), feature 02 (match exhaustiveness), feature 06 (closed-E totality),
etc. — *defer* with `… is not a struct declared in this file — completeness deferred`
Warnings instead of proving or disproving the property. The result the user reports: the
editor floods with deferral Warnings and the linting is effectively useless.

The cross-file and cross-package machinery **already exists** and is exercised by the
`goal check` CLI via `check.AnalyzePackageInDir(srcs, dir)`:
- `analyze.BuildPackage(srcs)` merges every file's name-keyed tables → cross-file resolution.
- `analyze.EnrichForeign(tables, srcs, dir, resolve)` parses imported Go packages and loads
  their exported struct field sets / function arities / methods → cross-package (non-goal)
  resolution.

The gap is purely in the editor path: the LSP does not feed the package's other files or
its directory into the checker. This feature closes that gap.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Cross-file goal types resolve in the editor (Priority: P1)

A developer edits `service.goal`, which contains `derive func toSpec(s Source) Spec`. The
target type `Spec` is declared in a sibling file `types.goal` in the same package directory.

**Why this priority**: This is the bulk of the reported noise — most deferrals are
same-package, cross-file references. Fixing this alone restores the usefulness of the
linting and is an independently shippable MVP.

**Independent Test**: Open a package with a `derive func` in one file and its target/source
struct in another; the editor shows no `completeness deferred` Warning for that type, and
shows a real Error if (and only if) a target field is genuinely unsourced.

**Acceptance Scenarios**:

1. **Given** `a.goal` declares `struct Spec { … }` and `b.goal` has a total
   `derive func` producing `Spec`, **When** the developer opens `b.goal`, **Then** no
   `unresolved-derive-type` / `completeness deferred` Warning is published for `Spec`.
2. **Given** the same package but the `derive func` omits a field that has no source and no
   override, **When** `b.goal` is open, **Then** an `unsourced-field` **Error** is published
   (the check is now proven, not deferred).
3. **Given** a struct literal `Spec{…}` in `b.goal` missing a required field declared in
   `a.goal`, **When** `b.goal` is open, **Then** a feature-08 `missing-field` Error is
   published instead of an `unresolved-literal-type` Warning.

### User Story 2 - Non-goal (Go-only) references resolve in the editor (Priority: P1)

A developer writes `derive func fromProto(p *pb.EnvironmentSpec) Spec` where `pb` is an
imported plain-Go package (standard library, same-module, or a third-party module).

**Why this priority**: The user explicitly called this out ("it should also handle
relationships into non-goal go only references"). The CLI already does this via
`EnrichForeign`; the editor must reach parity.

**Independent Test**: Open a file whose `derive func`/`from func` source or target is an
imported Go struct; the editor resolves the foreign struct's exported fields and no longer
defers, matching what `goal check` reports for the same file.

**Acceptance Scenarios**:

1. **Given** a `derive func` whose source is `*pb.Foo` from an imported Go package that
   resolves on disk, **When** the file is open, **Then** the foreign struct's exported
   fields are used to prove/disprove totality (no `completeness deferred` Warning for `pb.Foo`).
2. **Given** the import cannot be resolved (e.g. not yet `go get`-ed, network/toolchain
   unavailable), **When** the file is open, **Then** the type deferral Warning is shown (the
   honest "cannot tell here"), and no other diagnostics are lost.
3. **Given** a `recv.Method()?` whose receiver is an imported Go type, **When** the file is
   open, **Then** the feature-05 `?`-propagatability check resolves the foreign method the
   same way the CLI does.

### User Story 3 - Editor diagnostics match the CLI, with unsaved edits respected (Priority: P2)

A developer has unsaved changes in two open files of the same package.

**Why this priority**: Correctness/consistency. Diagnostics must reflect the in-editor
buffers (not stale disk contents) and must agree with `goal check` once saved. This prevents
a confusing class of "the editor says X but the CLI says Y" bugs.

**Independent Test**: With two open buffers in one package, edit a type in file A; the
deferral/Error state for a reference in file B updates to reflect A's *unsaved* text.

**Acceptance Scenarios**:

1. **Given** files A and B are both open in one package, **When** the developer edits A
   (fixing the type B references), **Then** B's published diagnostics update within one
   debounce interval — a reference that previously deferred now resolves or errors,
   reflecting A's *unsaved* text — without the developer touching B.
2. **Given** a file is analyzed, **When** the analysis completes, **Then** only the
   diagnostics belonging to the analyzed file are published to that file's URI (no sibling's
   diagnostics leak onto it).
3. **Given** the same package and no unsaved edits, **When** a file is analyzed in the
   editor and the same package is run through `goal check`, **Then** the two produce the same
   set of diagnostics for that file.

### Edge Cases

- **No resolvable directory** (non-`file:` URI, untitled buffer with no path, or a `file:`
  URI whose path does not exist as a directory — e.g. the test fixture `file:///x.goal`):
  fall back to single-file analysis (current behavior) rather than erroring or dropping
  diagnostics.
- **Analyzed file not yet on disk but its directory IS resolvable** (a new, never-saved
  `service.goal` in an existing package dir): do NOT fall back — run package analysis,
  overlaying the unsaved buffer as an extra source and tracking its index, so its
  cross-file references still resolve. (This is the boundary between FR-003 and FR-005.)
- **An open buffer that lives in a different directory** than the analyzed file must NOT be
  pulled into the package view (match files by directory, overlay by URI).
- **Valid `file:` URI with percent-encoding** (spaces, etc., common on macOS paths) must
  decode to the correct path; a decoding failure must not silently misresolve the directory.
- **Package discovery error** (two files in a directory declare different `package` names):
  the analyzed file must still get diagnostics — fall back to single-file analysis and log
  the discovery problem; never publish an empty/incorrect diagnostic set silently.
- **Foreign import resolution failure**: non-fatal — the specific type stays deferred; all
  other diagnostics for the file are still produced.
- **Performance under rapid edits**: building package tables and resolving Go imports
  (which may shell out to `go list`) must not run on every keystroke uncontrolled; the
  existing debounce plus reuse of already-open buffers must keep the editor responsive.
- **A directory with one open file and no siblings**: behaves exactly like today (package of
  one), plus foreign enrichment.
- **Diagnostics for a sibling file that is NOT open**: out of scope to publish (the editor
  owns which files show squiggles); the sibling's source is still *used* for resolution.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The language server MUST analyze an open `.goal` file using merged tables for
  its entire package (all `.goal` files sharing the file's directory), so that a type
  declared in a sibling file resolves instead of deferring.
- **FR-002**: The language server MUST enrich those tables with the exported declarations of
  imported Go packages referenced by `derive func` / `from func` signatures and
  `recv.Method()?` receivers, so non-goal (Go-only) references resolve — reaching parity with
  `check.AnalyzePackageInDir`.
- **FR-003**: When analyzing a file, the server MUST use the current in-editor buffer text
  for every open file in the package (including the file under analysis), and on-disk text
  for package files that are not open. Unsaved edits MUST be reflected. If the analyzed
  file's path is absent from the directory listing (new/unsaved), its buffer MUST be
  appended to the package sources and its index tracked, so it is still analyzed. The
  in-editor buffer is authoritative for any open file; `didSave` therefore requires no
  behavior beyond the existing change-driven analysis.
- **FR-003a**: The package view MUST include only open buffers whose resolved directory
  equals the analyzed file's directory — buffers open in other directories MUST NOT be
  pulled in. Files are matched by directory and overlaid by URI (buffer wins over disk).
- **FR-003b**: A `file:` document URI MUST be converted to its filesystem path correctly,
  including percent-decoding and the leading-slash form (`file:///abs/path.goal`). A URI
  that is not a usable `file:` path MUST trigger the FR-005 fallback, never a misresolved
  directory.
- **FR-004**: The server MUST publish to a file's URI only the diagnostics that belong to
  that file (correct per-file attribution from the package-level result).
- **FR-005**: When the open document cannot be mapped to a package **directory** on disk (no
  resolvable `file:` path, or the resolved directory does not exist), the server MUST fall
  back to single-file analysis so the file still receives diagnostics. The fallback condition
  MUST be logged to stderr, not surfaced as a user diagnostic. (Note: a resolvable directory
  whose analyzed file is merely unsaved is NOT a fallback case — see FR-003.)
- **FR-005a**: If the package directory resolves but its files cannot be read as a single
  coherent package (a sibling is unreadable, or two files declare conflicting `package`
  names — the same conflict `project.Discover` rejects), the server MUST fall back to
  single-file analysis for the analyzed file and log the reason to stderr. It MUST NOT
  publish a partial or incorrect diagnostic set.
- **FR-010**: When an open file changes (or opens/closes), the server MUST re-analyze every
  *other open file in the same package directory* — within the existing debounce/supersede
  model — and publish refreshed diagnostics for each, so a fix in one file clears or updates
  the stale cross-file diagnostics of its open siblings without the developer touching them.
  The package-level analysis returns per-file results in one pass, so one re-analysis can
  refresh all open siblings.
- **FR-006**: Foreign-import resolution failures MUST remain non-fatal: the unresolved type
  stays deferred (as today) while every other diagnostic for the file is still produced.
- **FR-007**: Editor diagnostics for a saved, conflict-free package MUST match the
  diagnostics `goal check` produces for the same file (same set of `Code`/`Pos`/`Severity`).
- **FR-008**: The IO the new path depends on (reading sibling files, resolving import paths
  to directories) MUST be injected behind an interface so the server is testable without the
  real filesystem or the `go` toolchain, consistent with the project's existing
  `analyze.DirResolver` seam and the repo's dependency-injection standard.
- **FR-009**: Package-table construction and foreign-import resolution (including any
  `go list` call) MUST occur at most once per debounce interval per scheduled analysis,
  never per keystroke, and MUST run off the protocol message-handling goroutine so the
  server keeps reading messages while analysis is in flight. Superseded (stale) analyses
  MUST be dropped (the existing `superseded` check). Directory-scoped caching of enriched
  tables is OPTIONAL and added only if measurement shows lag (see Open Questions).

### Key Entities

- **Open document set**: the server's existing `docs` map (URI → latest text + version);
  the source of truth for unsaved buffer contents.
- **Package view**: the set of `.goal` source strings for the open file's directory, formed
  by overlaying open buffers on top of discovered on-disk files.
- **Enriched tables**: `analyze.Tables` built from the package view and enriched with foreign
  Go declarations — the existing structure, just populated for the editor path.
- **DirResolver seam**: the existing `analyze.DirResolver` injected dependency mapping an
  import path to a package directory.
- **Resolver-injectable package-check entry**: a `check` function that composes
  `BuildPackage` + `EnrichForeign` + per-file run while accepting an injected
  `analyze.DirResolver` (the existing `AnalyzePackageInDir` hardcodes `nil`, so it cannot be
  faked in tests). The LSP supplies overlaid `srcs` + `dir` and receives `[][]Diagnostic`.
  Exact name/signature pinned in research.md and finalized in the plan step.
- **Directory-file reader seam**: an injected dependency that lists/reads the `.goal` files
  of one directory (cheaper than `project.Discover`'s whole-tree walk), faked in tests so no
  real disk is needed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For a representative same-package, cross-file reference, the number of
  `completeness deferred` / `unresolved-*` Warnings drops to zero where the type IS
  resolvable; deferrals remain only where the type genuinely cannot be read.
- **SC-002**: For a `derive func`/`from func`/`?` referencing a resolvable imported Go type,
  the editor produces the identical diagnostic set as `goal check` over the same package.
- **SC-003**: An edit to a buffer is reflected in cross-file diagnostics for other open files
  in the package within one debounce interval, using unsaved text.
- **SC-004**: The non-resolvable / off-disk / discovery-error paths each still publish the
  single-file diagnostic set (no regression to today's behavior, no lost diagnostics).
- **SC-005**: No `go list` / package-table build runs more than once per debounced analysis;
  the message loop never blocks on analysis (analysis runs off the handler goroutine).

## Testing Requirements *(mandatory)*

### Test Strategy

Tests use the Go standard-library `testing` package only — this project is zero-dependency
(no testify). Tests follow the existing `internal/lsp` and `internal/check` patterns:

- **LSP integration tests** (`internal/lsp`): drive the server with synthetic `didOpen` /
  `didChange` notifications across multiple URIs in one package and assert on the published
  `PublishDiagnostics` payloads. The server's IO seams (sibling-file reads, dir resolution)
  are injected with in-memory fakes so no real disk or `go` toolchain is needed — mirroring
  how `analyze.EnrichForeign` already takes an injectable `DirResolver` and how
  `server_test.go` already uses `file:///x.goal` with synchronous (`debounce<=0`) compiles.
- **Parity tests**: assert the editor path and `check.AnalyzePackageInDir` produce the same
  diagnostics for a fixture package (reuse/extend `internal/check/testdata` and
  `internal/analyze/testdata/extpkg`).
- **Fallback tests**: a non-`file:` URI and a package-discovery conflict each fall back to
  single-file analysis and still publish diagnostics.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration (lsp) | Cross-file `derive`/literal resolves; sibling-declared type no longer defers |
| FR-002 | Integration (lsp) | Imported Go struct/method resolves via injected resolver; no foreign deferral |
| FR-003 | Integration (lsp) | Unsaved edit in open file A changes diagnostics for open file B |
| FR-004 | Integration (lsp) | Published diagnostics are attributed to the correct file URI only |
| FR-005 | Integration (lsp) | Non-`file:` URI and absent file fall back to single-file analysis |
| FR-006 | Integration (lsp) | Unresolvable import defers that type only; other diagnostics intact |
| FR-007 | Parity | Editor path == `AnalyzePackageInDir` diagnostic set for a fixture package |
| FR-008 | Unit | IO seams are interfaces; server constructs with in-memory fakes |
| FR-009 | Integration (lsp) | Debounce coalesces rapid edits; stale (superseded) results dropped |
| FR-003a | Integration (lsp) | Open buffer in a different dir is excluded from the package view |
| FR-003b | Unit | `file:///abs/path%20with%20space/x.goal` decodes to the correct path |
| FR-005a | Integration (lsp) | Package-name conflict / unreadable sibling → single-file fallback + log |
| FR-010 | Integration (lsp) | Editing A refreshes open sibling B's diagnostics in one pass |

### Edge Case Coverage

- Off-disk / untitled / non-`file:` URI → single-file fallback test.
- Package-name conflict in directory → single-file fallback + stderr log test.
- Foreign import resolution failure → that type deferred, rest of diagnostics present.
- Single-file package (no siblings) → behaves like today plus enrichment.

## Out of Scope

- New diagnostic kinds or new completeness rules — this feature only makes EXISTING checks
  see cross-file and cross-package facts in the editor. (The deferred recursion classes the
  v1 checker intentionally keeps minimal — map/Option/nested — remain deferred.)
- Proto enum→sum bridging and oneof-wrapper modeling (explicitly excluded by
  `analyze/foreign.go` today).
- Multi-package / whole-workspace indexing beyond the open file's own package directory.
- Publishing diagnostics for files the editor has not opened.
- LSP features beyond diagnostics (hover, completion, go-to-definition).

## Open Questions

- **OQ-1 (perf/caching)**: Is a directory-scoped cache of enriched tables (invalidated on
  buffer change / save) needed for responsiveness, or does the existing 200ms debounce
  suffice for typical package sizes? Resolve by measurement during implementation; default to
  "no cache, rely on debounce" unless a representative package is visibly laggy. See
  research.md §Performance.
- **OQ-2 (foreign resolution on every compile)**: Should `go list`-backed foreign resolution
  be throttled separately from the goal-file debounce (it is the most expensive step)?
  Default: gate it behind the same debounce; revisit only if measured slow.
