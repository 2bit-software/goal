# Goal Expression and Pattern AST Nodes — Business Specification

## Overview

The goal AST must represent goal's own expression and pattern constructs, not
just the Go subset. This adds the node types for `match` (in both value and
statement position), destructuring patterns, postfix `?`, variant construction
with labeled arguments, and composite-literal spread. The central goal is that
the three meanings of `Enum.Variant(x)` — constructing a value, destructuring it
in a pattern, and an ordinary function call — become structurally distinct node
types, which is the structural fix for the current Match-before-Enums ordering
hack. This story defines the nodes and their traversal only; parsing and
lowering come in later stories.

## Functional Requirements

### FR-1: Match expression and arms
The AST SHALL provide a node for a `match` expression carrying its subject and
an ordered list of arms, usable in value position (e.g. `var x = match ...`) as
well as statement position. The AST SHALL provide a node for a single match arm
carrying its pattern and its body.

### FR-2: Destructuring and rest patterns
The AST SHALL provide a node for a destructuring variant pattern
(`Status.Active(a)`, including the data-less form `Status.Pending`) and a
distinct node for the rest/`_` catch-all arm pattern.

### FR-3: Postfix unwrap
The AST SHALL provide a node for the postfix `?` unwrap operator carrying its
operand.

### FR-4: Variant construction with labeled args
The AST SHALL provide a node for variant construction
(`Status.Active(since: now())`) carrying the enum, the variant tag, and its
arguments, and a node for a single labeled argument (`since: now()`).

### FR-5: Spread element
The AST SHALL provide a node for a composite-literal spread element
(`...defaults`, `...derive(s)`) carrying the spread expression.

### FR-6: Traversal
Every new node SHALL be traversable by the existing `Walk`/`Visitor` so that
`Walk` descends into each node's children exactly once, consistent with the
existing nodes.

## Acceptance Criteria

- [ ] The AST defines MatchExpr, MatchArm, VariantPattern, RestPattern,
      UnwrapExpr, VariantLit, LabeledArg, and SpreadElement.
- [ ] A test constructs a construction VariantLit and a destructuring
      VariantPattern of the same surface shape and asserts they are distinct
      node types.
- [ ] A test asserts `Walk` descends into the children of each new node exactly
      once (both the VariantLit and the VariantPattern walk correctly).
- [ ] `go build ./...`, `go vet ./...`, and `go test ./... -count=1` are green.

## User Interactions

None directly — these are internal compiler data structures consumed by the
parser, checker, backend, fmt, and LSP in later stories.

## Error Handling

Not applicable; node construction is by code. `Walk` on a nil node or nil child
field remains a no-op, consistent with existing behavior.

## Out of Scope

- Parsing these nodes from goal source (US-017 onward).
- Lowering / Go code emission for these constructs (US-033 onward).
- Exhaustiveness or other semantic checks over these nodes (US-029 onward).

## Open Questions

None. The node set and conventions are fixed by REWRITE-ARCHITECTURE.md §1.3 and
the existing internal/ast patterns.
