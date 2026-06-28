# LSP go-to-definition — Business Specification

## Overview

The goal language server SHALL answer `textDocument/definition` requests so an
editor user can jump from a referenced symbol to its declaration. The reference
under the cursor is resolved through the AST symbol graph of the open document
and the server returns the location of the declaration.

## Functional Requirements

### FR-1: Advertise the capability
The server SHALL advertise `definitionProvider` in its initialize response.

### FR-2: Resolve a function/method call
When the cursor is on the callee identifier of a call, the server SHALL return
the location of that function's (or method's) declaration name.

### FR-3: Resolve an enum variant
When the cursor is on an enum variant reference (in a construction value, a
match pattern, or an `Enum.Variant` selector), the server SHALL return the
location of that variant's declaration within its enum.

### FR-4: Resolve a type/enum name
When the cursor is on a reference to a top-level type, enum, or sealed-interface
name, the server SHALL return the location of that declaration's name.

### FR-5: Best-effort fallback
When the cursor is over no resolvable symbol, the document is not open, or the
source does not parse, the server SHALL return a null result (no error).

## Acceptance Criteria

- [ ] Initialize advertises `definitionProvider`.
- [ ] Definition of a function call resolves to the function declaration's name
      position.
- [ ] Definition of an enum variant resolves to the variant's declaration
      position inside its enum.
- [ ] Definition of a type/enum name reference resolves to its declaration.
- [ ] A position over whitespace or an unresolvable identifier yields null.
- [ ] An unknown document URI yields null and does not panic.

## User Interactions

The editor sends `textDocument/definition` with a document URI and a 0-based
line/character position. The server replies with a single `Location` (URI +
range) or `null`.

## Error Handling

All failure modes (unparseable source, unknown URI, cursor over nothing) return
`null` rather than a protocol error, matching the existing document-symbol and
semantic-token handlers.

## Out of Scope

- Cross-file / workspace-wide definition (definitions resolve within the open
  document's AST symbol graph for this milestone).
- Resolving local variables, parameters, and imported (qualified) symbols.
- Find-references and rename (US-050) and hover (US-049).

## Open Questions

- None. The acceptance criteria fully constrain the behavior.
