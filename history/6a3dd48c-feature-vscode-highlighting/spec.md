# Feature Specification: VS Code syntax highlighting (Layer 1)

**Feature**: vscode-highlighting
**Created**: 2026-06-25
**Status**: Complete
**Input**: "research what is required to make coloring/syntax highlighting work in VS Code for goal" → "let's build layer 1"

## User Scenarios

### User Story 1 - Open a `.goal` file and see it colored (Priority: P1)

A developer opens any `.goal` file in VS Code. The file is recognized as the Goal
language and syntax is colored: keywords, types, strings, comments, numbers, and
Goal-specific constructs are visually distinct.

**Independent Test**: Install the extension (or press F5 for the dev host), open
`editors/vscode/examples/sample.goal`, and observe coloring. Use "Developer:
Inspect Editor Tokens and Scopes" to confirm scopes.

**Acceptance Scenarios**:

1. **Given** a `.goal` file, **When** opened in VS Code, **Then** the language mode
   is "Goal" and Go-shared syntax (comments, strings, numbers, Go keywords) is colored.
2. **Given** Goal-specific syntax, **When** rendered, **Then** `enum`/`sealed`/`match`/
   `assert`/`from`/`derive`/`implements`, `Result`/`Option`, `=>`, postfix `?`,
   `...defaults`/`...derive`, and `///`+`>>>` doctests receive distinct scopes.

## Requirements

### Functional Requirements

- **FR-001**: The extension MUST register the `goal` language for the `.goal` extension.
- **FR-002**: The grammar MUST color Go's shared lexical structure (`//`, `/* */`,
  `"..."`, raw/rune literals, numbers, Go keywords, builtin types/functions).
- **FR-003**: The grammar MUST scope Goal keywords `enum`, `sealed`, `match`,
  `assert`, `from`, `derive`, `implements`.
- **FR-004**: The grammar MUST scope `Result`/`Option` as support types and their
  constructors (`Ok`/`Err`/`Some`/`None`) and `Enum.Variant` as enum members.
- **FR-005**: The grammar MUST scope the `=>` match arrow, postfix `?` unwrap, and
  `...defaults`/`...derive` spreads as distinct operators.
- **FR-006**: The grammar MUST treat `///` lines as documentation comments and
  highlight `>>>` doctest markers within them.
- **FR-007**: Language configuration MUST provide comment toggling, bracket
  matching, and auto-closing pairs.

### Key Entities

- **TextMate grammar** (`syntaxes/goal.tmLanguage.json`): scope name `source.goal`.
- **Language configuration** (`language-configuration.json`): editor behaviors.
- **Extension manifest** (`package.json`): language + grammar contributions.

## Success Criteria

- **SC-001**: All representative Goal tokens resolve to their intended scopes,
  verified programmatically with VS Code's tokenizer (19/19 assertions pass).
- **SC-002**: The extension packages to a valid `.vsix` with `vsce`.

## Testing Requirements

### Test Strategy

Grammar correctness is verified with `vscode-textmate` + `vscode-oniguruma` (the
exact engine VS Code uses) in `test/tokenize.test.mjs`, which tokenizes
`examples/sample.goal` and asserts token→scope mappings. Run via `npm test`.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-003 | Integration | `enum`/`match`/`assert`/`from`/`sealed`/`implements` get correct scopes |
| FR-004 | Integration | `Result`/`Option`/`Ok`/`None`/`State.Idle` get correct scopes |
| FR-005 | Integration | `=>`, `?`, `...defaults` get distinct operator scopes |
| FR-006 | Integration | `>>>` doctest marker scope inside `///` |
