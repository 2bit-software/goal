# Implementation Plan — US-017 Parse package, imports, declarations

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/parser/parser.go` | Hand-written recursive-descent parser: `ParseFile(src) (*ast.File, error)`, declaration tier (package/imports/type/var/const/func) + Go-subset type expressions + minimal operand expression parser + brace-span body skip. |
| `internal/parser/parser_test.go` | `package parser` test parsing a representative Go-subset sample and asserting the declaration-list shape (AC 2). |

### Modified Files
None. `internal/ast`, `internal/lexer`, `internal/token` already expose everything needed.

## Package Structure
```
internal/
  parser/
    parser.go        (new)
    parser_test.go   (new)
```
`internal/parser` imports `internal/lexer`, `internal/token`, `internal/ast`. No
import cycle: none of those import parser.

## Dependency Graph
1. token, lexer, ast (exist).
2. parser.go (depends on 1).
3. parser_test.go (depends on 2).

## Interface Contracts
```go
package parser

// ParseFile tokenizes src and parses it into an *ast.File. It returns the first
// parse error encountered (carrying a token.Pos) or nil for well-formed input.
func ParseFile(src string) (*ast.File, error)

// internal parser state
type parser struct {
    toks []token.Token // from lexer.Tokens, COMMENT/DOC_COMMENT skipped
    pos  int           // index of current token
    errs []error
}
```
Key internal methods: `cur()`, `at(k)`, `advance()`, `expect(k) token.Token`,
`errorf`, `parseFile`, `parsePackageClause`, `parseDecl`, `parseImportDecl`,
`parseGenDecl(tok)`, `parseValueSpec`, `parseTypeSpec`, `parseImportSpec`,
`parseFuncDecl`, `parseSignature`, `parseFieldList`, `parseType`, `parseTypeName`,
`parseStructType`, `parseInterfaceType`, `parseBlockSkip`, `parseExpr` (minimal
operand+postfix), `parseOperand`, `parseCompositeLit`.

## Integration Points
- Consumes `lexer.Tokens(src) []token.Token`.
- Produces `*ast.File` with `Decls []ast.Decl` of `*ast.GenDecl` / `*ast.FuncDecl`
  and `Imports []*ast.ImportSpec`.
- No existing caller is wired to the parser yet (US-026 introduces the new engine);
  this story is parser + test only.

## Testing Strategy
- `internal/parser/parser_test.go` (package parser, stdlib `testing` only).
- One happy-path test parses a multi-decl Go-subset sample (package, grouped+single+
  named+blank+dot imports, grouped const, grouped+single var, single+grouped type
  incl. a struct, and a method func with params/results/body) and asserts:
  file.Name.Name, len(file.Imports) and their paths/names, len(file.Decls), each
  decl's concrete type + GenDecl.Tok + spec names, and the func decl's name/recv/
  signature shape.
- Error-path tests: a malformed package clause and a stray token return non-nil error.
- Helper-level tests for type-expression parsing (pointer/slice/map/struct) if useful.
