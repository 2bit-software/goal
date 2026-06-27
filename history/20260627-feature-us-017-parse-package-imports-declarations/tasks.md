# Tasks — US-017 Parse package, imports, declarations

All tasks completed (parser.go + parser_test.go landed together as one cohesive
package; build/vet/test green).


## Task 1: Parser core + token cursor
- Create `internal/parser/parser.go` with the `parser` struct (tokens from
  `lexer.Tokens`, trivia skipped), cursor helpers (`cur`, `at`, `advance`,
  `expect`, `errorf`), and `ParseFile(src) (*ast.File, error)` parsing the package
  clause + an empty/decl loop scaffold.
- Covers: FR-1.
- Verify: `go build ./internal/parser`.

## Task 2: Type expressions + imports + gen decls
- Add `parseType` (qualified name, pointer, array/slice, map, struct, interface,
  func, chan, single index), `parseImportDecl`/`parseImportSpec`,
  `parseGenDecl`/`parseValueSpec`/`parseTypeSpec` (single + grouped), and the
  minimal operand+postfix `parseExpr` for initializer values.
- Covers: FR-2, FR-3, FR-5.
- Verify: `go build ./...`.

## Task 3: Function declarations + body skip
- Add `parseFuncDecl` (receiver, name, signature via `parseFieldList`) and
  `parseBlockSkip` (balanced-brace `*ast.BlockStmt`, positions set, List nil).
- Covers: FR-4.
- Verify: `go build ./...` && `go vet ./...`.

## Task 4: Tests
- Create `internal/parser/parser_test.go`: happy-path declaration-shape test over a
  representative Go-subset sample + error-path tests.
- Covers: all ACs.
- Verify: `go test ./... -count=1`.
