# Implementation Plan — US-024 Gate: parse 100% of corpus

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/corpus/parse_runner.go` | `Parser` interface + `ParserFunc` adapter + `RunParse(root, Case, Parser)` and an input-enumeration helper, mirroring runner.go / check_runner.go. |
| `internal/corpus/parse_runner_test.go` | `TestParseGate` — drives every unique `.goal` input in the manifest through `parser.ParseFile`; fails loudly listing each unparseable input. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/ast/ast.go` | Add `IndexListExpr{X, Lbrack, Indices []Expr, Rbrack}` (Pos/End/exprNode), parallel to go/ast, for multi-element index/type-arg lists. |
| `internal/ast/walk.go` | Add a Walk case for `*IndexListExpr` (descend X then each Index). |
| `internal/ast/ast_test.go` | Extend the Walk-descent coverage to include `IndexListExpr`. |
| `internal/parser/parser.go` | (1) `parseIndexSuffix`: parse a comma-separated list inside `[ ]`; 1 elt → `IndexExpr`, >1 → `IndexListExpr`. (2) `parseOperand`: accept type-literal starts (`[`, `map`, `struct`) so `[]byte(p)` conversions and `map[K]V{}` composite literals parse. (3) `compositeOK`: also allow `ArrayType`, `MapType`, `StructType`, `IndexListExpr`. |
| `internal/parser/goal_decl.go` | `parsePayloadField`: make the `:` optional (`name Type` and `name: Type` both parse). |
| `internal/parser/parser_test.go` (or a new `_test.go`) | Unit assertions for the three grammar additions (index list, `[]byte(p)`, optional-colon payload). |

## Package Structure

```
internal/
  ast/            # IndexListExpr node + Walk + test
  parser/         # grammar gap fixes + unit tests (package parser, internal)
  corpus/         # RunParse runner + whole-corpus gate test (package corpus)
corpus/manifest.json   # already present; consumed read-only by the gate
```

## Dependency Graph

1. `internal/ast` IndexListExpr + Walk (foundation; no parser dep).
2. `internal/parser` grammar fixes (depend on 1 for the new node).
3. `internal/corpus` RunParse + gate test (depend on 2 via `parser.ParseFile`).

No cycle: parser imports only lexer/token/ast; corpus may import parser since
nothing in corpus's existing deps (pipeline/check/project) imports parser.

## Interface Contracts

```go
// internal/ast
type IndexListExpr struct {
    X       Expr
    Lbrack  token.Pos
    Indices []Expr
    Rbrack  token.Pos
}
func (e *IndexListExpr) Pos() token.Pos { return e.X.Pos() }
func (e *IndexListExpr) End() token.Pos // Rbrack + 1
func (*IndexListExpr) exprNode() {}

// internal/corpus
type Parser interface{ Parse(src string) error }
type ParserFunc func(src string) error
func (f ParserFunc) Parse(src string) error { return f(src) }
// RunParse parses every .goal file backing the case (file-mode Input, or each
// Package.Files entry) and returns an error naming any input that fails.
func RunParse(root string, c Case, p Parser) error
```

The gate test wraps the parser as:
`corpus.ParserFunc(func(src string) error { _, err := parser.ParseFile(src); return err })`.

## Integration Points

- `internal/parser/parser.go::parseIndexSuffix` is the single index entry used by
  both `typeNameFrom` (type position) and `parsePostfix` (expression position),
  so widening it to a list fixes both contexts at once.
- `internal/parser/parser.go::parseOperand` feeds `parseExpr`, so accepting
  type-literal starts there makes index-list elements like `[]byte` parse
  uniformly in both positions.
- `internal/corpus/parse_runner.go` loads nothing new; the gate test reads
  `corpus.Load("../../corpus/manifest.json")` like the sibling runner tests.

## Testing Strategy

- `internal/ast/ast_test.go`: assert Walk descends into `IndexListExpr.X` and each
  `Indices` element exactly once (reuse the collector pattern).
- `internal/parser`: unit-parse `Result[int, error]` → IndexListExpr; `[]byte(p)`
  → CallExpr over ArrayType; `Foo { a int }` enum payload without colon.
- `internal/corpus/parse_runner_test.go`: `TestParseGate` enumerates every unique
  `.goal` input from the manifest, parses each, collects failures, and
  `t.Fatalf`/`t.Errorf`s with the full list if any fail (loud-failure AC). Loud
  zero-case guard (`t.Fatalf`) per the corpus-runner convention.
- Full project gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
