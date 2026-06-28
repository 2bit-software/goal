# Goal Declaration AST Nodes — Business Specification

## Overview

The goal AST front-end keeps one structural tree that every tool (parser,
checker, Go backend, fmt, LSP) traverses. The Go-subset nodes already exist; the
goal-specific *declarations* do not yet. This feature adds them so goal's closed
types and conversions are first-class in the tree rather than reconstructed by
token scanning.

The additions cover enum declarations (with and without payloads), sealed
interfaces, the struct `implements` clause, and the `from`/`derive` modifiers on
function declarations. Each new node participates in the shared pre-order
traversal so downstream tools can visit it uniformly.

## Functional Requirements

### FR-1: Enum declarations are representable
The AST SHALL provide an enum declaration node holding the enum's name and an
ordered list of variants. A variant SHALL hold its tag name and, optionally, a
payload — an ordered list of `name: Type` fields. A data-less variant SHALL be
distinguishable from a payload-bearing one.

### FR-2: Sealed interfaces are representable
The AST SHALL provide a sealed-interface declaration node holding the interface
name and its (possibly empty) method set.

### FR-3: The implements clause is representable
The AST SHALL provide an implements-clause node holding the named interface type
a struct is asserted to implement, attachable to a struct type.

### FR-4: from/derive modifiers are representable
A function declaration SHALL be able to record whether it is an ordinary
function, a `from` conversion, or a `derive` conversion, including the source
position of the modifier keyword. The reported start position of a modified
function SHALL be the modifier keyword.

### FR-5: Traversal descends into the new nodes
The shared tree-walk SHALL descend into the children of every new node: an enum
into its name and variants, a variant into its name and payload fields, a
payload field into its name and type, a sealed interface into its name and
methods, and an implements clause into its interface type.

## Acceptance Criteria

- [ ] The AST defines enum declaration, variant, and payload-field nodes.
- [ ] The AST defines a sealed-interface declaration node.
- [ ] The AST defines an implements-clause node.
- [ ] A function declaration can carry a from/derive modifier and its position.
- [ ] A test asserts the tree-walk descends into each new node's children.
- [ ] Project gates stay green: build, vet, and the full test suite pass.

## User Interactions

None directly. These are internal library types consumed by later parser,
checker, backend, and tooling stories.

## Error Handling

Not applicable: these are data types plus traversal cases. Nil children are
skipped by the existing traversal contract (a nil node is a no-op), matching the
established behavior of the Go-subset nodes.

## Out of Scope

- Goal expression and pattern nodes (match, postfix `?`, construction, spread) —
  US-016.
- Parsing goal source into these nodes — US-017 onward.
- A dedicated StructDecl node; the implements clause attaches to the existing
  struct type model.

## Open Questions

None.
