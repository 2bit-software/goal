# Business Spec — US-047 LSP semantic tokens

## Overview
The goal language server SHALL provide semantic token information so an editor
can highlight goal constructs by their semantic role (enum, enum variant, match,
postfix `?`, function, type, parameter, etc.) rather than by a generic grammar.

## Functional Requirements
- FR-1: On `initialize`, the server SHALL advertise a semantic-tokens capability
  including a legend (the ordered token-type and token-modifier names) and
  full-document support.
- FR-2: The server SHALL answer a `textDocument/semanticTokens/full` request for
  an open document with the document's semantic tokens, classified from the
  parsed AST.
- FR-3: Classification SHALL distinguish at least: keywords (including `match`,
  `enum`), enum type names, enum variant tags, the postfix `?` operator,
  functions/methods, type names, parameters, struct fields, strings, numbers,
  and comments.
- FR-4: Tokens SHALL be returned in document order, delta-encoded per the LSP
  semantic-tokens wire format.

## Acceptance Criteria
- [ ] The `initialize` response advertises the semantic-tokens provider (legend
      + full).
- [ ] A `textDocument/semanticTokens/full` request for an open document returns a
      non-empty, well-formed token array.
- [ ] A test asserts the classification of a sample containing an enum, a match,
      and a `?` expression: the enum name is classified as an enum, its variants
      as enum members, the `match` keyword as a keyword, and the `?` as an
      operator.
- [ ] An unparseable or unknown document yields an empty (non-nil) token set and
      never panics.

## User Interactions
- LSP client sends `textDocument/semanticTokens/full` with a document URI; the
  server replies with `{ data: [...] }`.

## Error Handling
- Unknown URI or source that does not parse: return empty token data (best
  effort), consistent with the existing document-symbol handler.

## Out of Scope
- Range requests (`/range`) and incremental delta (`/full/delta`).
- Full type resolution (hover/definition); unknown identifiers are simply not
  classified rather than guessed.

## Open Questions
- None blocking. Modifier set kept minimal (declaration) — purely additive.
