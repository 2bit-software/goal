# Business Spec — US-049 LSP hover with types

## Overview

The goal language server SHALL answer `textDocument/hover` requests so an editor
user hovering over a goal symbol sees a tooltip describing that symbol's type
(or signature) and any attached documentation.

## Functional Requirements

- FR-1: When the cursor is over a reference to, or the declaration of, a
  top-level goal symbol (function, method, enum, sealed interface, struct/type
  alias, or enum variant), the server SHALL return a hover describing that
  symbol.
- FR-2: For a function or method, the hover SHALL contain the symbol's
  signature, including its result type (e.g. a `Result[...]` result is shown
  verbatim) and any `///` doc-comment lines attached to the declaration.
- FR-3: The server SHALL advertise `hoverProvider` in its initialize
  capabilities.

## Acceptance Criteria

- AC-1: `textDocument/hover` returns the type and any doc comment for the symbol
  under the cursor.
- AC-2: Hover over a Result-returning function reports its signature.

## Error Handling

- A request for an unopened document URI SHALL yield a null hover.
- A cursor position over no resolvable symbol SHALL yield a null hover.
- Source that does not parse SHALL yield a null hover, never an error or panic.

## Out of Scope

- Cross-file / workspace-wide resolution (single open document only, like
  go-to-definition).
- Local variables, parameters, and qualified/imported symbols.
- Type inference of expression results beyond a declaration's written signature.
- Hover range echo-back (the optional `range` field of a Hover response).

## Open Questions

- None. The behavior mirrors the established best-effort, single-document LSP
  contract (go-to-definition, semantic tokens, document symbols).
