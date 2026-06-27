# Research — US-031

This is an internal reimplementation, not a greenfield design; the authoritative
references live in the repo, not on the web.

## Sources consulted (in-repo)

- `internal/check/mustuse.go` — the lexical must-use check (feature 03). Drop case:
  a statement-leading `IDENT(` to a `ModeResult`/`ModeResultClosed` callee with no
  trailing `?`. `_ :=` discard → Warning. Messages + codes to mirror.
- `internal/check/implements.go` — interface-satisfaction (feature 07). Folds
  embedded interfaces (`requiredMethods`), sealed = trivial, qualified/undeclared =
  defer Warning, signature compared by normalized `params|results` string.
- `internal/check/question.go` — open-E `?` arity/refusal (feature 05).
- `internal/check/closed.go` — closed-E `?` From-totality + `Result.Err` closedness
  (feature 06).
- `internal/sema/{check,fields}.go` — the established sema-check pattern (US-029/030):
  `visitorFunc` ast.Walk adapter, `Diagnostic` spine, message parity with the lexical
  checker, `exprName`/`quoteVariants` helpers already present.
- `internal/sema/resolve.go` + `sema.go` — `Info` model. Already carries
  `FuncSignatures` (Mode/E/Arity/EndsInError), `Enums` (VSet), `Methods` (Sig), `Sealed`,
  `FromRegistry`. **Missing**: in-file interface method sets — must be added.
- `internal/corpus/{sema_checker,sema_checker_test,sema_fields_test}.go` — the corpus
  runner-test pattern to copy for the 03/06/07 dirs.
- AST: `internal/ast/{ast,goal_expr,goal_decl}.go` and parser
  `parseMethodSpec`/`parseCallSuffix`/`makeVariantLit` — confirmed node shapes:
  `Result.Err(E.V)` is `CallExpr{Fun: SelectorExpr}`, a match-arm `Result.Err(b)` is a
  `VariantPattern` (distinct node), `f()?` is `ExprStmt{X: UnwrapExpr}`.

## Approach chosen

Read structure off the parse tree rather than re-lexing. The three meanings of
`Enum.Variant(x)` being distinct node types makes the legacy match-binding /
interface-method-vs-call-site false positives structurally impossible — no
brace-span or interface-span heuristics needed.

## Confidence

High. The behavior is pinned by the existing `testdata/check` `// want` markers and
the lexical checker is a line-by-line oracle for message wording.
