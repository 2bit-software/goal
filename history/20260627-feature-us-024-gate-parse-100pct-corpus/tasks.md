# Implementation Tasks — US-024

## Task 1: Add IndexListExpr AST node + Walk
**Status**: done
**Files**: internal/ast/ast.go, internal/ast/walk.go, internal/ast/ast_test.go
**Depends on**: (none)
**Spec coverage**: FR-4 (carrier for multi-element type-arg/index lists)
**Verify**: `go test ./internal/ast/ -count=1`

### Instructions
- In ast.go add `IndexListExpr{X Expr; Lbrack token.Pos; Indices []Expr; Rbrack
  token.Pos}` next to `IndexExpr`, with `Pos()` = X.Pos(), `End()` = Rbrack+1
  (offset/col +1 like IndexExpr.End), and `func (*IndexListExpr) exprNode() {}`.
- In walk.go add a `*IndexListExpr` case: walkExpr X, then walkExprList Indices.
- In ast_test.go extend the goal-expr/Walk-descent coverage to construct an
  IndexListExpr and assert Walk visits X and each Index once.

## Task 2: Close parser grammar gaps
**Status**: done
**Files**: internal/parser/parser.go, internal/parser/goal_decl.go,
internal/parser/parser_test.go
**Depends on**: Task 1
**Spec coverage**: FR-4
**Verify**: `go test ./internal/parser/ -count=1`

### Instructions
- parser.go `parseIndexSuffix`: after `[`, parse a comma-separated list of
  `parseExpr()` (exprLev++). One element → existing `*ast.IndexExpr`; more than
  one → `*ast.IndexListExpr{X, Lbrack, Indices, Rbrack}`. Used by both
  `typeNameFrom` (types) and `parsePostfix` (expressions), so both positions gain
  lists.
- parser.go `parseOperand`: add cases for `token.LBRACK`, `token.MAP`,
  `token.STRUCT` that return the parsed type (`parseArrayOrSliceType`,
  `parseMapType`, `parseStructType`) so `[]byte(p)` conversions and
  `map[K]V{}` / `[]T{}` composite literals parse. `parseExpr` then covers type
  args like `[]byte` uniformly.
- parser.go `compositeOK`: also return true for `*ast.ArrayType`, `*ast.MapType`,
  `*ast.StructType`, `*ast.IndexListExpr`.
- goal_decl.go `parsePayloadField`: consume the `:` only when present
  (`if p.at(token.COLON) { p.advance() }`) so `name Type` and `name: Type` both
  parse.
- parser_test.go: add focused parses — `Result[int, error]` yields IndexListExpr;
  `[]byte(p)` yields CallExpr over ArrayType; an enum variant `Foo { a int }`
  parses its payload field.

## Task 3: Add corpus parse runner + whole-corpus gate test
**Status**: done
**Files**: internal/corpus/parse_runner.go, internal/corpus/parse_runner_test.go
**Depends on**: Task 2
**Spec coverage**: FR-1, FR-2, FR-3
**Verify**: `go test ./internal/corpus/ -run TestParseGate -count=1`

### Instructions
- parse_runner.go: define `Parser interface{ Parse(src string) error }`,
  `ParserFunc`, and `RunParse(root string, c Case, p Parser) error` that parses
  the case's backing `.goal` file(s) — file-mode `Input`, or each `Package.Files`
  entry for package mode — returning an error naming any input that fails.
- parse_runner_test.go: `TestParseGate` (package corpus) loads
  `../../corpus/manifest.json` via `Load`, enumerates every unique input path
  across all cases, parses each through
  `ParserFunc(func(s string) error { _, err := parser.ParseFile(s); return err })`,
  collects failures, and `t.Errorf`s a full listing if any fail; `t.Fatalf` if
  the manifest yields zero inputs (loud zero-case guard, per US-003..US-008).

## Task 4: Run full project verify gates
**Status**: done
**Files**: (none — verification only)
**Depends on**: Task 3
**Spec coverage**: AC build/vet/test
**Verify**: `go build ./... && go vet ./... && go test ./... -count=1`

### Instructions
- Run the three prd.json verifyCommands. If the gate surfaces a parse failure not
  covered by Tasks 1-2, return to Task 2, close the gap, and re-run until every
  corpus input parses and all three gates are green.
