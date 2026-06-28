# Business Spec — US-018 Parse statements

## Goal

As a compiler engineer, I need statement parsing so function bodies become a real
AST statement list (today they are captured as an opaque balanced-brace block with
no contents).

## Functional Requirements

- The parser SHALL turn a function body into an ordered list of statement nodes.
- The parser SHALL handle: ordinary assignment, short-variable declaration,
  if (with optional init and else/else-if), for (infinite, condition-only,
  three-clause, and range), expression switch (with case and default clauses),
  return, defer, go, break/continue, increment/decrement, nested blocks, and
  const/var/type declarations used as statements.
- A statement list SHALL be delimited structurally (the lexer inserts no
  semicolons), and a control-clause body brace SHALL NOT be misread as a
  composite literal.

## Acceptance Criteria

1. Parser handles assignment, short-var, if/for/switch/return/defer/go and block
   statements.
2. A test parses a function body with each statement form and asserts the
   statement list.

## Out of Scope

- Precedence-correct expression parsing and postfix `?` (US-019).
- goal-specific statements: match/patterns/construction (US-021/022).
- Semicolon insertion / newline-aware lexing.
