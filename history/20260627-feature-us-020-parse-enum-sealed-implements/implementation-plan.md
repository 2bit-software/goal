# Implementation Plan — US-020 Parse enum, sealed, implements

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/parser/goal_decl.go` | Parser methods for the goal closed-type declarations: `parseEnumDecl`, `parseVariant`, `parsePayloadField`, `parseSealedInterfaceDecl`, plus the implements-clause helper. Keeps goal-specific parsing separate from the Go-subset `parser.go`. |
| `internal/parser/goal_decl_test.go` | Tests parsing the `features/01-enums` and `features/07-implements` example inputs and asserting variant/field/implements structure. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/parser/parser.go` | `parseDecl` dispatches `token.ENUM` -> `parseEnumDecl` and an IDENT `"sealed"` -> `parseSealedInterfaceDecl`. `parseStructType` consumes an optional `implements <TypeName>` (contextual IDENT) into `StructType.Implements` before the field-list brace. |

## Package Structure

```
internal/parser/
  parser.go         (modified: dispatch + struct implements)
  goal_decl.go      (new: enum / sealed-interface parsing)
  parser_test.go    (unchanged)
  goal_decl_test.go (new: corpus-driven structural test)
```

## Dependency Graph

1. `ast` nodes + `token` kinds — already exist (US-011, US-015). No work.
2. `internal/parser/goal_decl.go` — new parse methods, depends only on existing
   `ast`/`token` and the helpers in `parser.go` (`ident`, `expect`, `parseType`,
   `parseInterfaceType` internals).
3. `internal/parser/parser.go` edits — wire dispatch + struct implements; depends
   on (2) for the method names.
4. `internal/parser/goal_decl_test.go` — depends on (2) and (3).

## Interface Contracts

```go
// in goal_decl.go (methods on *parser)
func (p *parser) parseEnumDecl() *ast.EnumDecl
func (p *parser) parseVariant() *ast.Variant
func (p *parser) parsePayloadField() *ast.PayloadField
func (p *parser) parseSealedInterfaceDecl() *ast.SealedInterfaceDecl

// helper used by parseStructType in parser.go
func (p *parser) parseImplementsClause() *ast.ImplementsClause // nil if absent
```

`isContextualKeyword(t token.Token, word string) bool` — true when `t.Kind ==
token.IDENT && t.Lit == word`; used to recognize `sealed`/`implements`.

## Integration Points

- `parser.go::parseDecl` switch: add `case token.ENUM: return p.parseEnumDecl()`
  and a leading check `if p.at(token.IDENT) && p.cur().Lit == "sealed" { return
  p.parseSealedInterfaceDecl() }`.
- `parser.go::parseStructType`: after `kw := p.expect(token.STRUCT)` and before
  `p.expect(token.LBRACE)`, call `st.Implements = p.parseImplementsClause()`
  (which returns nil and consumes nothing when the next token is not the
  contextual `implements`).

## Testing Strategy

`goal_decl_test.go` (package `parser`, internal — same as `parser_test.go`):
- `TestParseEnumDecl`: read `features/01-enums/examples/status.goal`, parse,
  locate the `*ast.EnumDecl`, assert variant names, that `Pending` is data-less,
  and that `Cancelled` has payload fields `reason string` and `at Time`.
- `TestParseSealedInterface`: read `features/01-enums/examples/shape.goal`, find
  the `*ast.SealedInterfaceDecl` named `Shape` with an empty method set, and
  assert the `Circle`/`Rectangle` structs carry an `Implements` clause naming
  `Shape`.
- `TestParseImplements`: read `features/07-implements/examples/*.goal`, assert the
  struct's `Implements.Type` is the expected `*ast.Ident` / `*ast.SelectorExpr`
  (`io.Writer` for the qualified case).
- All tests assert `ParseFile` returns no error for these inputs.

Test inputs are read from the repo via the relative path `../../features/...`
(tests run with cwd = `internal/parser`).
