# Audit: AI-Consumer Readiness — US-016

## Findings

### None CRITICAL / None MAJOR
The spec is implementable without guessing: each node maps to a concrete struct
following the established `internal/ast` conventions (token.Pos fields,
Pos()/End(), category marker, a Walk case). The technical-requirements-research
file already lists field names per node. Acceptance criteria 2 and 3 are
directly translatable into Go test assertions (dynamic-type inequality and a
Walk visit-count of 1 per child).

### MINOR — "distinct node types" assertion mechanism unspecified
The spec says assert two nodes are "distinct node types" but not how. A standard
Go approach (compare `fmt.Sprintf("%T", ...)`, or a type switch) is obvious and
non-ambiguous; no clarification needed.

## Assumptions

- New nodes live in a new file `internal/ast/goal_expr.go`, paralleling
  `goal_decl.go`, to keep the Go subset (ast.go) and goal additions separated.
- The new test lives in `internal/ast/ast_test.go` (or a sibling `_test.go` in
  package ast) reusing the existing collector/assertChildren helpers.
- Verify gates are the prd.json verifyCommands; no new tooling is introduced.
