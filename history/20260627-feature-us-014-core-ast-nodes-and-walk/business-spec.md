# Core AST Nodes and Walk — Business Specification

## Overview
This feature introduces `internal/ast`, the single structural model of goal
source that every later tool (parser, checker, Go backend, LSP, fmt) will
traverse. It delivers the core Go-subset node types — the declarations,
statements, expressions, and type expressions that goal shares with Go — plus a
position-carrying `Node` interface and a `Walk(Visitor, Node)` pre-order
traversal. Goal-specific nodes (enums, match, patterns, construction) are added
by later stories on top of this skeleton.

## Functional Requirements

### FR-1: Position-carrying Node hierarchy
The package SHALL define a `Node` interface exposing source positions, plus
category marker interfaces (`Decl`, `Stmt`, `Expr`) that embed `Node`, so tools
can switch on node category. Every concrete node SHALL report a `token.Pos`.

### FR-2: File and declaration nodes
The package SHALL define `File` (package name, imports, top-level declarations)
and the Go declaration nodes goal uses: a general declaration covering
import/const/var/type with its specs (import spec, value spec, type spec) and a
function declaration.

### FR-3: Statement nodes
The package SHALL define the Go statement nodes goal uses, including block,
expression, assignment/short-var, return, if, for, range, switch with its case
clauses, defer, go, branch (break/continue), declaration, and inc/dec
statements.

### FR-4: Expression and type-expression nodes
The package SHALL define the Go expression nodes (identifier, basic literal,
parenthesized, unary, binary, selector, index, slice, call, star/pointer,
composite literal, key-value, function literal) and the type-expression nodes
(array/slice, map, struct, interface, function, channel, ellipsis) together with
field-list and field nodes for parameters, results, and struct members.

### FR-5: Walk traversal with Visitor
The package SHALL define `Visitor` (a `Visit(Node) Visitor` contract) and
`Walk(Visitor, Node)` that performs a pre-order traversal: it visits a node, and
when the visitor returns non-nil, recurses into that node's children and then
signals end-of-children. Each node in a tree SHALL be visited exactly once per
traversal.

## Acceptance Criteria

- [ ] `internal/ast` defines `File`, the Go decl/stmt/expr/type nodes goal uses,
      and `Walk(Visitor, Node)`.
- [ ] Every concrete node type satisfies a `Node` interface that reports a source
      position.
- [ ] A unit test builds a tree by hand (covering a representative spread of node
      types) and asserts `Walk` visits every node exactly once.
- [ ] `go build ./...`, `go vet ./...`, and `go test ./... -count=1` are all
      green.

## User Interactions
None directly. This is an internal compiler library consumed by the parser and
the downstream tools in later stories.

## Error Handling
`Walk` on a nil node is a no-op (it does not panic). The node types are plain
value/struct containers; construction validity is the parser's concern, not this
package's.

## Out of Scope
- Goal-specific declaration nodes (enum, sealed interface, implements, from/derive
  modifiers) — US-015.
- Goal-specific expression/pattern nodes (match, variant pattern, unwrap, variant
  literal, spread) — US-016.
- The parser that builds these nodes from tokens — US-017+.
- Rendering/printing the AST — later stories (US-025/US-032/US-045).

## Open Questions
None. The node taxonomy follows the well-established `go/ast` model trimmed to
the subset goal uses, with our own `token.Pos`.
