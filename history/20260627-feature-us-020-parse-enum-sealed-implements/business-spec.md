# Parse enum, sealed, implements — Business Specification

## Overview

The goal parser must recognize the language's closed-type declaration surface so
that source files using enums, sealed interfaces, and `implements` clauses parse
into a faithful AST instead of failing. This is the declaration-tier slice of
the new front-end that makes closed types first-class structure rather than
token-splice artifacts.

## Functional Requirements

### FR-1: Enum declarations
The parser SHALL parse an `enum Name { ... }` declaration into an enum
declaration node whose variants appear in source order. It SHALL accept
data-less variants (a bare tag) and payload variants (a tag followed by a brace
block of `name: Type` fields), with multiple payload fields separated by commas.

### FR-2: Sealed interface declarations
The parser SHALL parse a `sealed interface Name { ... }` declaration into a
sealed interface node, including the common empty-body form `{}` and a body
containing method specifications.

### FR-3: Struct implements clause
The parser SHALL parse a struct type declared as
`type T struct implements Iface { ... }`, attaching the named interface
(an unqualified name such as `Shape` or a qualified name such as `io.Writer`)
to the struct type's implements clause, then parsing the field list normally.

## Acceptance Criteria

- [ ] An enum with only data-less variants parses with each variant recorded as
      a bare tag and no payload.
- [ ] An enum with payload variants parses with each payload field's name and
      type recorded, including a variant carrying multiple comma-separated fields.
- [ ] A `sealed interface Name {}` parses into a sealed interface node with an
      empty method set.
- [ ] A `type T struct implements Iface { ... }` parses with the implements
      clause carrying the interface name and the field list parsed as usual.
- [ ] The qualified implements form (`implements io.Writer`) parses with the
      clause carrying a qualified name.
- [ ] A test parses the `features/01-enums` and `features/07-implements` example
      `.goal` inputs and asserts the variant / field / implements structure.
- [ ] `go build ./...`, `go vet ./...`, and `go test ./... -count=1` all pass.

## User Interactions

None directly user-facing. Consumers call `parser.ParseFile(src)` and receive an
`*ast.File` whose declarations now include enum, sealed-interface, and
implements-bearing struct nodes.

## Error Handling

Malformed closed-type declarations record a parse error (surfaced via the joined
error from `ParseFile`) while always advancing the cursor so parsing makes
progress, consistent with the existing parser's error discipline.

## Out of Scope

- Match expressions / patterns (US-021).
- Construction, labeled args, spread (US-022).
- `from`/`derive`, assert, doctests (US-023).
- Any lowering, code emission, or semantic checking of these constructs.

## Open Questions

None.
