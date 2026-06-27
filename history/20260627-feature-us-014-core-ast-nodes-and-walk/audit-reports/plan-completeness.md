# Plan Audit: Coverage — US-014

## Findings

### None CRITICAL / None MAJOR
- FR-1 (Node hierarchy + positions) → ast.go interfaces with Pos()/End(). Covered.
- FR-2 (File + decls) → File, GenDecl/FuncDecl, Import/Value/TypeSpec. Covered.
- FR-3 (statements) → full Stmt list. Covered.
- FR-4 (exprs + type exprs + fields) → full Expr list + Field/FieldList. Covered.
- FR-5 (Walk + Visitor) → walk.go. Covered.
- Acceptance test → ast_test.go counting visitor. Covered.

### MINOR — End() beyond strict spec
Plan adds `End()` to Node; spec only requires `Pos()`. This matches go/ast and
costs little. Not scope creep that affects acceptance.

## Assumptions
- `End()` included on Node for go/ast parity and future fmt/LSP ranges.
- Private marker methods used to close category interfaces to the package.
