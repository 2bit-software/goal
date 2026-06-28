# Business Spec — US-025 Add AST snapshot tests

## Outcome

A compiler engineer can see, and is alerted to any change in, the structural
shape the parser produces for each goal construct.

## Requirements

- The AST can be rendered to a stable, human-readable textual form that is
  deterministic for a given tree.
- A checked-in snapshot exists for one representative input per goal construct
  (enum, match/patterns, postfix `?`, closed-Result match, implements/sealed,
  `...defaults` spread, assert, doctests, from/derive).
- A test parses each representative input, renders its AST, and fails loudly if
  the rendering differs from the checked-in snapshot.

## Constraints

- Zero third-party dependencies; stdlib `testing` only (no testify).
- The textual form must omit byte positions so it pins structure, not offsets.
