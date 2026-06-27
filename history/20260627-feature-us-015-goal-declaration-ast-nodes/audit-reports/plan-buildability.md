# Plan Audit — Buildability

## Checks

- **Dependency order valid**: token → core ast nodes → goal_decl.go → walk.go
  cases → test. No forward references; each layer compiles on top of the one
  above. Valid topological order.
- **Interface contracts agree**: new nodes reuse existing types (`*Ident`,
  `Expr`, `*FieldList`) and the `declNode()` marker, so they satisfy `Decl`/
  `Node` without changing those interfaces. `FuncMod` is a new local type with
  no external coupling.
- **File paths verified**: `internal/ast/{ast.go,walk.go,ast_test.go}` exist;
  `internal/ast/goal_decl.go` does not yet exist (no conflict). Confirmed no
  consumers of ast.FuncDecl/StructType exist outside internal/ast, so adding
  fields is non-breaking (all existing literals are keyed).
- **Integration points specific**: StructType.Implements is the exact attach
  point; walk.go's `*StructType` case is the exact edit site; new switch cases
  named per node.

## Findings

### MINOR-1: Keyed-literal assumption
Adding fields to FuncDecl/StructType is safe only because all existing
constructions are keyed struct literals. Verified true in ast_test.go (the sole
consumer). Documented so a future positional literal would be caught. Not
blocking.

## Verdict

No CRITICAL or MAJOR. The plan is buildable in the stated order with each step
compiling. Recommend PASS.

## Assumptions

- `go build`/`go vet`/`go test ./...` is the full gate; the corpus and other
  packages are unaffected since the change is additive within internal/ast.
- FuncDecl.Pos() returning ModPos when modified matches the leading-keyword Pos
  convention used by the other decl nodes.
