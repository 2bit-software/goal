# Implementation Plan — US-013

## File Inventory

### New Files
None. All target token kinds already exist in `internal/token` (US-011).

### Modified Files
| File | Changes |
|------|---------|
| `internal/lexer/lexer.go` | Emit goal lexemes: add `?` → QUESTION; match `=>` → FAT_ARROW in the `=` case; match `...` → ELLIPSIS in the `.` case; detect `///` → DOC_COMMENT in the `/` case via a new `scanDocComment` helper. Update the package doc comment (remove the US-012 "deferred to US-013" caveats). |
| `internal/lexer/lexer_test.go` | Add tests asserting each lexeme yields exactly one token of the expected kind, DOC_COMMENT vs COMMENT distinctness, and implements/sealed/from/derive → IDENT. |

## Package Structure

```
internal/
  token/      (unchanged — kinds already defined)
  lexer/
    lexer.go        (modified)
    lexer_test.go   (modified)
```

## Dependency Graph

1. `internal/token` — already complete (QUESTION, FAT_ARROW, ELLIPSIS, DOC_COMMENT).
2. `internal/lexer/lexer.go` edits — depend on 1.
3. `internal/lexer/lexer_test.go` edits — depend on 2.

## Interface Contracts

No public API change. `lexer.New`, `(*Lexer).Next() token.Token`, and
`lexer.Tokens(src string) []token.Token` keep their signatures. New unexported
helper:

```go
// scanDocComment consumes a /// doc comment to end of line, emitting DOC_COMMENT.
func (l *Lexer) scanDocComment(pos token.Pos) token.Token
```

Behavioral contracts (token streams, EOF elided):
- `"?"`   → `[QUESTION]`
- `"=>"`  → `[FAT_ARROW]`
- `"..."` → `[ELLIPSIS]`
- `"/// d"`→ `[DOC_COMMENT("/// d")]`
- `"// d"` → `[COMMENT("// d")]`
- `"implements"` → `[IDENT("implements")]` (same for sealed/from/derive)

## Integration Points

All within `internal/lexer/lexer.go`:
- `scanOperator` `case '/'`: if `peek()=='/'` then check a third `/` (one rune
  past peek) to route to `scanDocComment`; else keep `scanLineComment`.
- `scanOperator` `case '='`: peek for `>` → FAT_ARROW before the existing
  `op2(ASSIGN, '=', EQL)`.
- `scanOperator` `case '.'`: peek for `..` (two more dots) → ELLIPSIS before the
  existing single PERIOD. (The `.`-then-digit FLOAT branch in `Next()` is
  unaffected: `...` has no digit after the first dot.)
- Add `case '?'`: consume one rune → QUESTION.
- A second-rune lookahead beyond `peek()` is needed for `///` and `...`; add a
  small `peek2()` helper or inline-decode the rune after `rdOffset`.

## Testing Strategy

Extend `internal/lexer/lexer_test.go` (package `lexer`, stdlib `testing` only,
no testify — matches existing file). Table-driven `TestGoalLexemes` over
`{src, wantKind, wantLit}` rows asserting `Tokens(src)` is `[wantKind, EOF]`.
Separate `TestDocCommentVsComment` asserting `///` → DOC_COMMENT and `//` →
COMMENT with retained Lit. `TestContextualKeywordsAreIdent` over
implements/sealed/from/derive → IDENT. Reuse the existing helpers/style in the
file. Verify gates: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
