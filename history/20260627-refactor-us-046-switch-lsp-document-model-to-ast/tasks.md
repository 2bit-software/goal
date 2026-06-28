# Implementation Tasks — US-046

## Task 1: Derive document symbols from the AST
**Status**: completed
**Files**: internal/lsp/symbols.go
**Depends on**: (none)
**Spec coverage**: FR-1, FR-2, FR-3; AC "scanDecls removed", outline kinds,
bodyless non-overrun, empty/partial robustness, handler.
**Verify**: `go test ./internal/lsp -run 'Symbol|DocumentSymbol' -count=1`

### Instructions
- Rewrite `collectSymbols(src string) []DocumentSymbol` to parse with
  `parser.ParseFile(src)`. On error (or nil File), return `[]DocumentSymbol{}`.
- Walk `file.Decls`. For each declaration produce zero or more symbols:
  - `*ast.EnumDecl` -> kind symEnum, name d.Name, range d.Pos()..d.End().
  - `*ast.SealedInterfaceDecl` -> symInterface, detail "sealed interface",
    name d.Name, range d.Pos()..d.End().
  - `*ast.GenDecl` with `d.Tok == token.TYPE` -> one symbol per `*ast.TypeSpec`:
    kind by spec: `spec.Assign != (token.Pos{})` -> symClass; `spec.Type` is
    `*ast.StructType` -> symStruct; `*ast.InterfaceType` -> symInterface; else
    symClass. Range start = `d.Pos()` (the `type` keyword) when `len(d.Specs)==1`
    else `spec.Pos()`; range end = `spec.End()`. SelectionRange = name.
  - `*ast.FuncDecl` -> symMethod if `d.Recv != nil` else symFunction; name
    d.Name; range d.Pos()..d.End(). (from/derive already report symFunction and
    d.Pos() points at the modifier keyword.)
  - Any other decl kind, or a nil/empty name Ident -> skip (best-effort).
- Reuse `rangeOf(src, startOff, endOff int) Range` for both ranges, passing
  `node.Pos().Offset` / `node.End().Offset`.
- Delete the `decl` struct, `scanDecls`, `skipLine`, `declEnd`, and the
  `goal/internal/scan` import. Add `goal/internal/parser`, `goal/internal/ast`,
  `goal/internal/token` imports as needed.
- Keep `documentSymbols` and `rangeOf` unchanged.

## Task 2: Widen diagnostic ranges via the AST front-end lexer
**Status**: completed
**Files**: internal/lsp/diagnostics.go
**Depends on**: (none)
**Spec coverage**: FR-4 (no LSP-local token scanner).
**Verify**: `go test ./internal/lsp -count=1`

### Instructions
- In `tokenEnds(text string) map[int]int`, replace `scan.Lex(text)` with
  `lexer.Tokens(text)`. For each token `t`, set `m[t.Pos.Offset] = t.Pos.Offset
  + len(t.Lit)`.
- Replace the `goal/internal/scan` import with `goal/internal/lexer`.
- Leave `toLSP`, `compile`, `compileSingle`, and the rest unchanged.

## Task 3: Verify gates and grep guards
**Status**: completed
**Files**: (none)
**Depends on**: Task 1, Task 2
**Spec coverage**: AC "full LSP suite passes", AC "scanDecls removed".
**Verify**:
- `go build ./...`
- `go vet ./...`
- `go test ./... -count=1`
- `grep -rn 'scanDecls' internal/lsp` returns nothing.
- `grep -rn 'internal/scan' internal/lsp/*.go` (non-test) returns nothing.

### Instructions
- Run the verify gates and grep guards; all green / empty before completion.
