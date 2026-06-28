# Technical Requirements / Research — US-031

## Where

- `internal/sema/mustuse.go` — new `CheckMustUse(*ast.File, *Info)`.
- `internal/sema/implements.go` — new `CheckImplements(*ast.File, *Info)`; extend
  `internal/sema/resolve.go` + `sema.go` `Info` with in-file interface method sets
  (`Interfaces map[string][]Method`) and embeddings (`EmbeddedIfaces map[string][]string`).
- `internal/sema/question.go` — new `CheckQuestion` (open-E feature 05) and
  `CheckClosed` (closed-E feature 06), keyed off each plain func's resolved Mode.
- `internal/sema/check.go` — wire the four new checks into `sema.Check`.
- `internal/corpus/sema_question_test.go` (or similar) — corpus runner test over
  the 03-result, 06-error-e, 07-implements dirs via `SemaCheck`.

## Key AST facts (correct-by-construction vs the token scanner)

- A dropped Result is `*ast.ExprStmt{X: *ast.CallExpr{Fun: *ast.Ident}}` to a func
  whose `Info.FuncSignatures[name].Mode` is `ModeResult`/`ModeResultClosed`. A `?`
  consumes it because `f()?` parses as `ExprStmt{X: *ast.UnwrapExpr{...}}` (X is the
  UnwrapExpr, not the CallExpr).
- A `_ :=` discard is an `*ast.AssignStmt` whose single Lhs is Ident `_` and Rhs is
  a CallExpr — advisory Warning.
- `Result.Err(E.Variant)` is a `*ast.CallExpr{Fun: *ast.SelectorExpr(Result.Err)}`
  with a `*ast.SelectorExpr` (or `*ast.VariantLit`) argument. A match-arm
  `Result.Err(b) =>` is a `*ast.VariantPattern`, a different node — so the
  construction walk never mis-reads a binding (legacy false positive impossible).
- Interface method specs: `parseMethodSpec` yields a `*ast.Field` with `Names=[name]`
  + `Type=*ast.FuncType` for a method, or `Names=nil` + `Type=Ident` for an embedded
  interface. Signature normalization must reuse resolve.go's `paramTypeListFL` /
  `joinTypes` so interface method `Sig` matches concrete method `Sig` exactly.
- `?` statement context (assign vs bare, discard vs bound) is read from the enclosing
  `*ast.AssignStmt`/`*ast.ExprStmt`; only those well-formed positions are checked.

## Message parity

Mirror `internal/check/{mustuse,implements,question,closed}.go` message wording so
the inline `// want` markers pass. Diagnostic codes reused verbatim:
`dropped-result`, `unresolved-result-discard`, `unimplemented-method`,
`method-signature-mismatch`, `unresolved-interface`, `missing-from-conversion`,
`err-outside-closed-enum`, `unknown-error-variant`, `unresolved-err-value`,
`unresolved-error-enum`, `unresolved-question-error`, plus the feature-05 question codes.
