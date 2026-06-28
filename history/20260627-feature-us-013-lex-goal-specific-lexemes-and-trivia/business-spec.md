# US-013 Lex goal-specific lexemes and trivia — Business Specification

## Overview

The goal lexer must recognize the language's distinctive lexemes as single,
correctly-typed tokens, so downstream tools (parser, fmt, LSP) consume them
directly rather than re-splicing source text. This story extends the existing
core-Go-subset lexer to emit the goal-specific operators and the doc-comment
trivia whose token kinds were already defined in US-011.

## Functional Requirements

### FR-1: Postfix unwrap operator
A `?` in the source SHALL produce exactly one QUESTION token.

### FR-2: Match-arm fat arrow
A `=>` in the source SHALL produce exactly one FAT_ARROW token — never an
ASSIGN token followed by a GTR token.

### FR-3: Ellipsis
A `...` in the source SHALL produce exactly one ELLIPSIS token, distinct from a
sequence of PERIOD tokens.

### FR-4: Doc-comment trivia
A `///` and the rest of its line SHALL produce exactly one DOC_COMMENT token
whose literal text is retained verbatim. An ordinary `//` line comment SHALL
remain a COMMENT token, and a `/* */` block SHALL remain a COMMENT token.

### FR-5: Contextual keywords are identifiers
The words `implements`, `sealed`, `from`, and `derive` SHALL each lex as an
IDENT token, not as a reserved keyword.

## Acceptance Criteria

- [ ] `?` lexes to a single QUESTION token (stream is QUESTION, EOF).
- [ ] `=>` lexes to a single FAT_ARROW token, not ASSIGN + GTR.
- [ ] `...` lexes to a single ELLIPSIS token, not three PERIODs.
- [ ] `/// doc` lexes to a single DOC_COMMENT token retaining its text.
- [ ] `// note` still lexes to a COMMENT token (distinct from DOC_COMMENT).
- [ ] Each of implements/sealed/from/derive lexes to a single IDENT token.
- [ ] The existing US-012 core-subset lexing is unchanged (no regressions).

## User Interactions

None directly. The lexer is an internal compiler component; behavior is
observed through its emitted token stream and exercised by unit tests.

## Error Handling

Unchanged from US-012: unrecognized bytes still yield ILLEGAL tokens and the
cursor still advances to guarantee progress. `?`, `=>`, `...`, and `///` are no
longer unrecognized and therefore no longer emit ILLEGAL / mis-typed tokens.

## Out of Scope

- Attaching comments to specific AST nodes (parser-layer trivia attachment).
- Any parser, AST, or backend work — this story is lexer-only.
- Adding implements/sealed/from/derive as reserved keywords (they stay IDENT).

## Open Questions

None. The token kinds and lexer seam already exist; this is a closed,
fully-specified extension.
