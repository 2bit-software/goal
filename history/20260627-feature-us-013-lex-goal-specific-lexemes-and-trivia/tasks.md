# Tasks — US-013

## Task 1: Emit goal lexemes and DOC_COMMENT in the lexer
- **Files**: `internal/lexer/lexer.go`
- **Changes**:
  - Add `peek2()` (rune two positions ahead) to support `///` and `...`.
  - `scanOperator` `case '?'`: consume one rune → QUESTION.
  - `case '='`: if next rune is `>` → FAT_ARROW (one token); else existing
    ASSIGN/EQL handling.
  - `case '.'`: if next two runes are both `.` → ELLIPSIS; else existing single
    PERIOD.
  - `case '/'`: if `//` and a third `/` follows → `scanDocComment` (DOC_COMMENT);
    else existing `//` COMMENT / `/* */` COMMENT handling.
  - Add `scanDocComment` helper (mirrors `scanLineComment`, tags DOC_COMMENT,
    retains full `/// ...` text).
  - Update package doc comment to drop the US-012 "deferred to US-013" caveats.
- **Spec coverage**: FR-1, FR-2, FR-3, FR-4. (FR-5 already satisfied by
  existing `scanIdentifier`/`token.Lookup`.)
- **Verify**: `go build ./...` succeeds.

## Task 2: Tests for goal lexemes, trivia, and contextual keywords
- **Files**: `internal/lexer/lexer_test.go`
- **Changes**:
  - `TestGoalLexemes`: table over `?`→QUESTION, `=>`→FAT_ARROW, `...`→ELLIPSIS,
    each asserting `Tokens(src)` == `[kind, EOF]` (and not the old split forms).
  - `TestDocCommentVsComment`: `/// d` → single DOC_COMMENT with retained Lit;
    `// d` → single COMMENT (distinct).
  - `TestContextualKeywordsAreIdent`: implements/sealed/from/derive each → IDENT.
- **Spec coverage**: FR-1..FR-5 (acceptance-criteria assertions).
- **Verify**: `go vet ./...` and `go test ./... -count=1` are green.

## Status
- Task 1: completed
- Task 2: completed

## Dependency order
Task 1 → Task 2 (tests assert the behavior Task 1 introduces).
