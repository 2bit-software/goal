# Research — US-014 Core AST nodes and Walk

## Question
What node set and Walk traversal should `internal/ast` define for the Go subset?

## Authoritative reference: go/ast
The Go standard library `go/ast` is the canonical, battle-tested model and the
architecture doc (REWRITE-ARCHITECTURE.md §3) explicitly says the parser is
"Go-grammar-shaped". Mirroring go/ast's node taxonomy and its `Walk`/`Visitor`
contract is the lowest-risk, most idiomatic choice and keeps the later parser
(US-017..023) and Go backend (US-032) on familiar ground.

### Node interface + marker interfaces
go/ast uses `Node{ Pos(); End() }` plus three category markers `Expr`, `Stmt`,
`Decl`, each embedding Node with a private marker method. We adopt the same so
backends/parser can switch on category. We already have `token.Pos`
(Offset/Line/Col) and `token.Token`, so positions are first-class.

### Walk/Visitor contract (go/ast convention)
`Walk(v Visitor, node Node)`: call `v.Visit(node)`; if it returns a non-nil
Visitor `w`, recurse into the node's children with `w`, then call `w.Visit(nil)`
to signal end-of-children. A node is *visited once* per the non-nil
`v.Visit(node)` call — this is exactly what the acceptance test counts.

## Scope decision
US-015 (enum/sealed/implements/from-derive decls) and US-016 (match/pattern/
construction exprs) add the goal-specific nodes. US-014 is strictly the Go
subset: File, GenDecl/FuncDecl, ImportSpec/ValueSpec/TypeSpec, the Go
expr/type/stmt nodes, FieldList/Field. Keep the package open for those additions
(Walk's type switch will gain cases later).

## Alternatives considered
- **Reuse go/ast directly.** Rejected: goal is not a clean Go superset; the tree
  must carry goal-specific nodes (US-015/016) and our own token.Pos. The doc
  (§2) rejects reusing go/parser/go/ast for the same reason.
- **Flat/offset-keyed model (today's scan+analyze).** Rejected: this whole
  rewrite exists to replace it with one real tree.

## Confidence: High
go/ast is a proven blueprint; the only project-specific shaping is using our
token.Pos and reserving room for goal nodes.

## Next steps
Plan the concrete node list + Walk; implement in internal/ast with a
hand-built-tree Walk test that counts every node exactly once.
