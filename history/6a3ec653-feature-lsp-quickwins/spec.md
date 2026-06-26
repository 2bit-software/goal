# Feature Specification: High-ROI LSP quick wins

**Initiative**: `6a3ec653-feature-lsp-quickwins`
**Created**: 2026-06-26
**Status**: Draft
**Input**: "automode, no branching, build 1-3 for this" — the three highest-ROI LSP features
identified for the goal language server: (1) an idiomatize-file code action backed by
`internal/fix`, (2) precise diagnostic ranges, (3) document symbols (outline).

## Context

The `goal` language server (`goal lsp`, used by the VSCode extension) is diagnostics-only
today: it advertises only `textDocumentSync` and handles `initialize`/`shutdown`/`exit` plus
the `didOpen`/`didChange`/`didClose` document-sync notifications. This feature adds three
capabilities chosen for high value relative to effort, each reusing infrastructure that
already exists (`internal/fix.File`, `internal/scan` token offsets, `check.OffsetToPosition`).

None of the three requires the "declaration position index" that go-to-definition/hover would
need, so they are independently shippable without that larger investment.

## User Scenarios & Testing *(mandatory)*

### User Story 1 — One-click "idiomatize this file" (Priority: P1)

A developer has a `.goal` file containing plain-Go patterns (`(T, error)` signatures,
`if err != nil` propagation, `switch` that should be `match`). They want to convert it to
idiomatic goal in one action, including on save.

**Why this priority**: The rewrite engine (`internal/fix.File`) is already written, tested,
and proven safe (it refuses rewrites it can't prove). Exposing it is the highest
value-to-effort ratio on the board — the editor gains an automatic "fix all" with almost no
new logic.

**Independent Test**: Open a file with a fixable plain-Go pattern; trigger the source/fix-all
action (or save with fix-all-on-save); the document is replaced with the idiomatic rewrite.
A file with nothing to fix offers no action and is left untouched.

**Acceptance Scenarios**:

1. **Given** a file whose source `internal/fix.File` rewrites, **When** the editor requests
   code actions for it, **Then** the server returns a `source.fixAll`-kind action carrying a
   workspace edit that replaces the document with the rewritten source.
2. **Given** a file `internal/fix.File` leaves unchanged, **When** code actions are requested,
   **Then** no fix-all action is offered (no empty/no-op edit).
3. **Given** the fix-all action is applied, **When** the edit lands, **Then** the resulting
   document equals `internal/fix.File`'s output for that source, and re-running yields no
   further change (idempotent — the fixer runs to a fixed point).
4. **Given** the server advertises its capabilities, **When** the client initializes,
   **Then** the capabilities include a code-action provider declaring the fix-all kind.
5. **Given** a fixable file, **When** code actions are requested with `context.only=[]` and
   separately with `only=["source.fixAll"]`, **Then** the fix-all action is returned in both;
   **When** requested with `only=["quickfix"]`, **Then** it is not returned.
6. **Given** a fixable pattern inside an otherwise syntactically broken file, **When** the
   code action is requested, **Then** the server returns a safe edit or none — never a panic.

### User Story 2 — Diagnostics underline the exact construct (Priority: P1)

A developer sees a diagnostic squiggle and expects it to cover the offending token/construct,
not run to the end of the line.

**Why this priority**: Trivial change, immediately visible on every diagnostic the editor
already shows. Today the range spans finding→end-of-line because "no token length is
available," but the token's end offset is right there in the lexer output.

**Independent Test**: Produce a known diagnostic; assert its published range ends at the
offending token's end, not the line end.

**Acceptance Scenarios**:

1. **Given** a diagnostic located at a token, **When** it is converted to a protocol
   diagnostic, **Then** the range spans from the token's start to the token's end.
2. **Given** a diagnostic whose offset does not correspond to a lexable token start (a
   defensive edge), **When** converted, **Then** it falls back to today's end-of-line span
   rather than producing an empty or invalid range.
3. **Given** multi-line source, **When** ranges are computed, **Then** start and end
   positions remain correct 0-based line/character values.

### User Story 3 — Outline / symbol navigation (Priority: P2)

A developer opens a `.goal` file and wants the outline pane, breadcrumb bar, and
"Go to Symbol in Editor" (⌘⇧O) populated with the file's declarations.

**Why this priority**: Among the most-used editor features, and it needs only the open file's
top-level declaration positions — no cross-file resolution, no new tables. Slightly more code
than #1/#2 (a dedicated lexical pass), so P2.

**Independent Test**: Open a file with assorted declarations; request document symbols; assert
each top-level declaration appears once with the correct name, kind, and range.

**Acceptance Scenarios**:

1. **Given** a file declaring enums, structs, interfaces, type aliases, functions, methods,
   and `from`/`derive func`s, **When** document symbols are requested, **Then** each is
   returned with its name, an appropriate symbol kind, and a range covering its declaration.
2. **Given** an enum with variants (or a struct with fields), **When** symbols are requested,
   **Then** the variants/fields MAY appear as nested children of the enum/struct symbol
   (decision in research; at minimum the top-level symbol is present).
3. **Given** the server advertises capabilities, **When** the client initializes, **Then** a
   document-symbol provider is declared.
4. **Given** a syntactically messy or partial file, **When** symbols are requested, **Then**
   the server returns the symbols it can locate without erroring (best-effort, never a crash).

### Edge Cases

- **Fix-all on a file with unsaved edits**: the action operates on the current buffer text
  the server holds, not stale disk content.
- **Fix-all needs no package context**: `internal/fix.File` is a single-file transform; the
  action uses only the open buffer.
- **Code action / document symbol requested for an unknown/closed URI**: return an empty
  result, not an error.
- **Diagnostic at end of file / zero-length token**: range falls back safely (US2 #2).
- **Unfixable candidates** (`internal/fix` `Skip`/`Warn` reports): see Open Questions —
  whether/where to surface them.
- **Request arrives for a non-`file:` or untitled buffer**: features operate on buffer text
  the server holds; no filesystem access is required for any of the three.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The server MUST handle `textDocument/codeAction` and, for a buffer that
  `internal/fix.File` would rewrite, return a `source.fixAll.goal`-kind code action titled
  **"Idiomatize file (goal fix)"** whose workspace edit replaces the whole document with the
  rewritten source. No action is returned when the rewrite is a no-op.
- **FR-001a**: The code action's edit MUST be **version-pinned**: it carries the source
  buffer's version (via `documentChanges` → `TextDocumentEdit` with an
  `OptionalVersionedTextDocumentIdentifier`), so the client rejects it if the buffer changed
  between request and apply (no clobbering of concurrent typing). The replaced range MUST span
  the exact extent of that same buffer (`[0,0]`..end-of-buffer).
- **FR-001b**: The server MUST honor `context.only`: return the fix-all action only when
  `only` is empty OR a requested kind is a prefix of the action's kind (`source`,
  `source.fixAll`, or `source.fixAll.goal`). This makes it surface both on manual Quick Fix
  and via `editor.codeActionsOnSave`.
- **FR-002**: The server MUST advertise a code-action provider (declaring the fix-all kind) in
  its initialize capabilities, so the editor offers the action and can run it on save.
- **FR-003**: The fix-all edit MUST be computed from the server's current buffer text for the
  URI and MUST equal `internal/fix.File`'s output, so applying it is idempotent.
- **FR-004**: Diagnostic ranges MUST span the offending token (its start offset to its end
  offset, both converted independently via `OffsetToPosition` so a multi-line token's end is
  correct). The range MUST fall back to the current finding→end-of-line behavior when the
  finding's offset is not a lexed token start OR the resolved end is ≤ the start. Positions
  remain 0-based line/character.
- **FR-005**: The server MUST handle `textDocument/documentSymbol` and return the open file's
  top-level declarations as hierarchical `DocumentSymbol[]`, each with a name, the LSP symbol
  kind below, a `range` (declaration keyword → closing `}`, or signature span if bodyless),
  and a `selectionRange` (the name token span, ⊆ range). It MUST advertise a document-symbol
  provider. Kind mapping (normative): enum→Enum, struct→Struct, interface & `sealed interface`
  →Interface, type alias→Class, plain func→Function, method (func with receiver)→Method,
  `from func`/`derive func`→Function.
- **FR-006**: Document-symbol extraction MUST be best-effort and tolerant of partial/invalid
  source — it returns the symbols it can locate and never errors or panics.
- **FR-007**: Code-action and document-symbol requests for an unknown or closed URI MUST
  return an empty result (not an error), and MUST NOT block the message loop. The wire shape
  for "no result" MUST be an empty JSON array `[]`, never `null`.
- **FR-008**: These additions MUST NOT regress existing diagnostics behavior (single-file and
  the cross-file/cross-package path) or the document-sync notifications.
- **FR-009**: The code-action handler MUST be best-effort and tolerant of partial/invalid
  source — it MUST NOT panic; on broken source it returns either a safe rewrite edit or no
  action (mirrors FR-006 for symbols). Neither the code-action nor symbol handler may touch
  `analysisMu`, so they cannot deadlock with diagnostics compilation.
- **FR-010**: As the deliverable is inert until deployed, completion MUST rebuild and install
  the `goal` binary to GOBIN (`task install`) and the user MUST reload the VSCode window to
  load the new server; confirm the extension's `package.json` needs no change (it forwards
  requests via vscode-languageclient). The editor-population success clauses (SC-003 outline)
  are verified manually post-reload.

### Key Entities

- **Idiomatize rewrite**: `internal/fix.File(src) → (out, changes, reports)`; the code action
  wraps `out` in a full-document workspace edit when `out != src`.
- **Token offsets**: `scan.Lex` tokens (`Start`/`End`) drive precise diagnostic ranges.
- **Symbol**: a top-level declaration with name, kind, and range, derived by a lexical pass
  over `scan.Lex` tokens + `check.OffsetToPosition`.
- **Protocol surface**: new request handlers (`codeAction`, `documentSymbol`) and new
  capability advertisements, layered onto the existing generic JSON-RPC request/response.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Triggering fix-all on a file with a known plain-Go pattern produces a document
  identical to `internal/fix.File`'s output; re-triggering yields no further edit.
- **SC-002**: Every published diagnostic's range covers its offending token (verified against
  the token's end offset), not the whole line — except the documented fallback.
- **SC-003**: For a fixture file, the `documentSymbol` response includes every top-level
  declaration exactly once with the correct kind/range (asserted in test). Editor outline /
  ⌘⇧O population is verified manually after install + reload (FR-010).
- **SC-004**: Existing LSP tests and the full `go test ./...` suite remain green.

## Testing Requirements *(mandatory)*

### Test Strategy

Stdlib `testing` only (zero-dependency project; no testify). Reuse the `internal/lsp` test
patterns (synchronous server `debounce<=0`, framed-message capture, the `NewServerWithIO`
harness). Drive `textDocument/codeAction` and `textDocument/documentSymbol` requests and
assert on the JSON-RPC responses; unit-test the range mapping and symbol extraction directly.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration (lsp) | codeAction on a fixable file returns a fixAll WorkspaceEdit; no-op file returns none |
| FR-002 | Integration (lsp) | initialize advertises the code-action provider/kind |
| FR-003 | Unit/Integration | fix-all edit equals `fix.File` output; idempotent |
| FR-004 | Unit | toLSP range uses token end; fallback for non-token offset |
| FR-005 | Integration (lsp) | documentSymbol returns each top-level decl with name/kind/range |
| FR-006 | Unit | symbol extraction on partial source returns best-effort, no panic |
| FR-007 | Integration (lsp) | codeAction/documentSymbol for unknown URI → empty result |
| FR-008 | Integration (lsp) | existing diagnostics tests still pass |

### Edge Case Coverage

- No-op fix-all → no action. Unknown URI → empty result. EOF/zero-length token → range
  fallback. Partial source → best-effort symbols.

## Out of Scope

- Go-to-definition, hover, find-references, rename (need the declaration position index — a
  separate, larger investment).
- Semantic tokens / semantic highlighting (TextMate grammar already covers basics).
- gofmt-style formatting (`internal/fix` is idiomatization, not layout formatting).
- Workspace-wide / unopened-file diagnostics.
- `workspace/executeCommand` plumbing — the idiomatize action is delivered as a code action
  with a workspace edit, which covers on-save and manual trigger without a command round-trip.

## Open Questions

- **OQ-1 (unfixable-candidate reports)**: `internal/fix.File` returns `reports` for candidates
  it detected but refused to rewrite (Skip/Warn). Surface them as Information/Hint diagnostics?
  If so, compute them on `didSave` only (not per keystroke) to bound cost. **Default**: defer
  to a follow-up; v1 ships the rewrite action only. Decide in plan.
- **OQ-2 (symbol nesting)**: Return enum variants / struct fields as nested child symbols, or
  only top-level declarations in v1? **Default**: top-level first; nest children if cheap.
- **OQ-3 (fix-all kind & on-save)**: RESOLVED — kind `source.fixAll.goal`, surfaced when
  `context.only` is empty or prefix-matches `source`/`source.fixAll`/`source.fixAll.goal`
  (FR-001b). Extension `package.json` needs no change (FR-010).
