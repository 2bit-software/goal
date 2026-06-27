# US-046 Switch LSP document model to AST — Business Specification

## Overview

The goal LSP currently builds its document outline (textDocument/documentSymbol)
and widens diagnostic ranges using a bespoke token scan (`internal/scan`). The
project is migrating every tool onto the shared AST front-end. This story moves
the LSP document model onto that AST: document symbols are derived by parsing the
file and walking its declarations, and the LSP no longer carries its own token
scanner. Behavior visible to an editor (symbol kinds, ranges, selection ranges,
and diagnostics) is unchanged.

## Functional Requirements

### FR-1: Document symbols derived from the AST
The file outline is produced by parsing the document and walking the parsed
declarations. Each top-level declaration is reported once with the correct
symbol kind: enum, struct, interface, sealed interface (interface kind), type
alias (class kind), function, method, and `from`/`derive func` (function kind).

### FR-2: Faithful ranges
Each symbol's full range covers its declaration; its selection range covers the
declaration's name. A bodyless declaration (type alias, `from`/`derive func`)
does not extend its range over the declaration that follows it.

### FR-3: Best-effort robustness
Source that does not parse (partial or malformed) yields an empty, non-nil
outline rather than an error or panic. Empty source yields an empty outline.

### FR-4: No LSP-local token scanner
The LSP no longer depends on the legacy token scanner for its document model or
for widening diagnostic ranges; range widening uses the AST front-end's lexer.
Diagnostic findings continue to come from the existing checker.

## Acceptance Criteria

- [ ] The document outline reports every top-level declaration form with the
      expected symbol kind (enum, struct, interface, sealed interface, alias,
      function, method, from/derive func).
- [ ] A bodyless alias or from/derive func does not swallow the next declaration.
- [ ] Each symbol's selection range starts at or after its full range start.
- [ ] Empty source yields a non-nil empty outline; malformed source does not panic.
- [ ] The document-symbol request returns the outline for an open document and an
      empty result for an unknown document.
- [ ] The `scanDecls` token walk is removed from the LSP document-symbol code.
- [ ] The full existing LSP test suite passes.

## User Interactions

Editor LSP requests: `textDocument/documentSymbol` (outline) and
`textDocument/publishDiagnostics` (findings). Contracts are unchanged.

## Error Handling

Unparseable or partial buffers produce an empty outline; an unknown document URI
produces an empty outline. Diagnostics for a superseded revision are dropped as
before.

## Out of Scope

- Migrating the checker (`internal/check`) itself off the legacy analyze tables;
  the LSP keeps consuming `check.Analyze` / package analysis (which the LSP
  package path needs for foreign type resolution).
- New LSP features (semantic tokens, go-to-definition, hover, rename) — later
  stories US-047..US-050.

## Open Questions

- None. The existing LSP test suite is the behavioral contract.
