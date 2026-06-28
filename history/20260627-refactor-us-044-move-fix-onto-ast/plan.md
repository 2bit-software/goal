# Implementation Plan — US-044 Move goal fix onto the AST

## Goal

Replace internal/fix's token-scan detection (`scan.Lex` + `analyze.Build` +
`analyze.FuncSpans`/`SigAt`/`ScanFuncs`) with AST-driven detection
(`parser.ParseFile` + `sema.Resolve`), keeping the same minimal `scan.Replacement`
byte edits and the same reports. Public API unchanged.

## Files

- `internal/fix/fix.go` — `File` loop: per-iteration `parser.ParseFile(out)` +
  `sema.Resolve(file)` + a `typeDecls` map walked from the AST; dispatch the four
  fixers over the parsed tree; keep `scan.Splice` to apply edits; keep helpers
  (lineOf, indentOf, lineStartBefore, spanHasComment). On parse error, return the
  source unchanged for that iteration (conservative no-op).
- `internal/fix/resultsig.go` — convert `(T, error)` FuncDecl results to
  `Result[T, error]` via `FuncType.Results` (FieldList Opening/Closing);
  classify each top-level `ReturnStmt` (skip nested FuncLit) into success
  (`return v, nil` → `Result.Ok(v)`) / bare propagation / non-conforming.
- `internal/fix/propagate.go` — value-binding form (consecutive `AssignStmt`+
  `IfStmt`) and init-guard form (`IfStmt.Init`), emitting `... := rhs?` / `rhs?`;
  Option deref rewrites (`*o` StarExpr → `o`).
- `internal/fix/match.go` — `SwitchStmt` whose first `CaseClause` label is
  `Enum.Variant` in `info.Enums` → match suggestion.
- `internal/fix/callsite.go` — `IfStmt{Init:nil, Cond: v != nil}` in a
  ModeNone function, preceded by an `AssignStmt` whose last LHS == v → call-site.

## Risks

- Every input and every post-splice intermediate must parse; verified against
  the fix test suite. Parse failure falls back to a no-op iteration.
- Comments are dropped by the parser → keep `spanHasComment` as a raw byte scan.

## Verification

`go build ./...`, `go vet ./...`, `go test ./... -count=1` (esp.
`./internal/fix` and `./internal/lsp`).
