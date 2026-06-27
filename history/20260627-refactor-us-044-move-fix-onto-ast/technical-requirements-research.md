# Technical Requirements / Research — US-044

## Approach

Drive candidate detection off the parsed tree:

- Parse with `parser.ParseFile(out)` → `*ast.File` each loop iteration.
- Resolve name-keyed facts with `sema.Resolve(file)` → `*sema.Info`
  (FuncSignatures carry Mode: ModeNone/ModeResult/ModeResultClosed/ModeOption,
  and T the success type). This replaces `analyze.Build`'s token scan.
- Build a `typeDecls map[string]string` (name → "struct"/"interface"/underlying)
  by walking `file.Decls` TypeSpecs, to feed `analyze.ZeroLit` for the
  zero-value comparison in propagation/return classification.

## Emitting edits

Keep emitting minimal `scan.Replacement` byte edits and applying them with
`scan.Splice` — these are pure string utilities, not lexing, and they preserve
untouched formatting/comments exactly (the AST drops comments and would reflow
if printed). Byte ranges come from AST node `Pos()/End()` offsets
(`token.Pos.Offset`).

## Node mapping

- Function signature: `FuncDecl.Type.Results` (a `*ast.FieldList` with
  `Opening`/`Closing` paren positions; unparenthesized single result has zero
  Opening). Convert only an all-unnamed parenthesized `(T, error)` tuple.
- Propagation value-binding form: consecutive block statements
  `AssignStmt(:=)` then `IfStmt{Init:nil, Cond: BinaryExpr(v != nil / v == nil)}`
  whose body is exactly one valid propagation `ReturnStmt`, no `Else`.
- Init-guard form: `IfStmt{Init: AssignStmt(condVar := call), Cond: condVar != nil}`.
- Valid propagation returns: `Result.Err(condVar)` (CallExpr over
  SelectorExpr), `zero, condVar`, `Option.None` (SelectorExpr), `nil`.
- Option deref: `*o` parses as a deref expr; rewrite each to `o`, abort if the
  pointer is used in any other shape.
- switch-over-enum: `SwitchStmt` whose first `CaseClause`'s first label is
  `Enum.Variant` (SelectorExpr) with Enum in `info.Enums`.
- call-site: `IfStmt{Cond: v != nil}` whose binding's last LHS name is the
  cond var, inside a function whose sema Mode is ModeNone.

## Comment safety

`spanHasComment(src, lo, hi)` stays a raw byte-substring scan over the source
range — comments are invisible to the AST, so this guard is still needed to
refuse collapses that would drop a `//`/`/*` comment.

## Caveat

The parser must successfully parse every input and every intermediate
(post-splice) state in the fix fixed-point loop. If a parse fails, fix returns
the source unchanged for that iteration (no-op), matching the conservative
contract. Verified against the fix test suite inputs.
