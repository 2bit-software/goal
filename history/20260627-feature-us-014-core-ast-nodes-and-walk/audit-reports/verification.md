# Verification — US-014

## Gates (all green)
- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./... -count=1` — all packages pass, incl. new `goal/internal/ast`

## Acceptance criteria → evidence
1. "internal/ast defines File, the Go decl/stmt/expr/type nodes goal uses, and
   Walk(Visitor,Node)." → internal/ast/ast.go (File + GenDecl/FuncDecl +
   Import/Value/TypeSpec + Field/FieldList + the Go expr/type/stmt node set) and
   internal/ast/walk.go (Visitor + Walk). Compiles and vets clean.
2. "A test builds a tree by hand and asserts Walk visits every node exactly
   once." → internal/ast/ast_test.go: TestWalkVisitsEveryNodeExactlyOnce builds
   a representative *File tree, registers every constructed node, and asserts
   each is visited exactly once (visit count map) AND no extra node is visited.
   TestWalkNilNodeIsNoop covers the nil no-op. Both pass.

## Findings
None. Story complete; committed as c42a8ad.
