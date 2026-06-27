# Plan Audit — Coverage

## Trace: spec acceptance criteria → plan

| Acceptance criterion | Plan element |
|----------------------|--------------|
| Enum decl/variant/payload-field nodes | `goal_decl.go`: EnumDecl, Variant, PayloadField |
| Sealed-interface decl node | `goal_decl.go`: SealedInterfaceDecl |
| Implements-clause node | `goal_decl.go`: ImplementsClause + StructType.Implements |
| from/derive modifier + position | `ast.go`: FuncDecl.Mod/ModPos + FuncMod enum |
| Test: Walk descends into each new node's children | `ast_test.go`: TestWalkGoalDeclChildren |
| Build/vet/test green | verifyCommands |

Every functional requirement (FR-1..FR-5) maps to at least one plan element. No
unmapped plan elements (no scope creep): the only edits to existing nodes are
the two additive fields directly required by FR-3 and FR-4.

## Findings

### MINOR-1: End() positions are untested
The plan tests traversal and modifier behavior but not `End()` values of the new
nodes. Impact: low — Pos/End follow the established mechanical convention and are
exercised indirectly; the story's acceptance is about Walk descent, not exact
End offsets. Not blocking.

## Verdict

No CRITICAL or MAJOR. Recommend PASS.

## Assumptions

- ImplementsClause is hosted on StructType rather than a new StructDecl.
- Variant/PayloadField/ImplementsClause are support nodes (Node only), matching
  Field/FieldList.
