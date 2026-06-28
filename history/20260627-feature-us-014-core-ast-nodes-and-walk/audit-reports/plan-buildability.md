# Plan Audit: Buildability — US-014

## Findings

### None CRITICAL / None MAJOR
- Dependency order is valid: token (exists) → ast.go → walk.go → test. No forward
  references.
- Interface contracts agree: Visitor.Visit(Node) Visitor; Walk type-switches on
  concrete node pointers and recurses via walkList/nil-guards — standard go/ast
  shape, known to compile.
- File paths are concrete (internal/ast/{ast.go,walk.go,ast_test.go}); the dir is
  new (confirmed absent).
- Test is buildable: package ast, stdlib testing only, hand-built tree + counting
  visitor; the once-per-node property is the go/ast Walk invariant.

### MINOR — Exact field shapes finalized at implement time
The precise field set of each node (e.g. SliceStmt vs SliceExpr indices) is
settled during implementation against go/ast; immaterial to acceptance (count
test is structure-agnostic).

## Assumptions
- Walk counts a node on the non-nil Visit(node) call; the Visit(nil) end-markers
  are not counted.
- Optional child fields (e.g. IfStmt.Init/Else, FuncDecl.Recv) are nil-guarded.
