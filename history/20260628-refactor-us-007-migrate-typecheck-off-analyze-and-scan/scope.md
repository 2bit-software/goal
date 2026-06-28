# Scope — US-007

## What is being refactored and why

internal/typecheck (the depth checker) still depends on internal/analyze
(analyze.Tables / analyze.BuildPackage / analyze.ModeResult) and the
internal/scan token lexer (scan.Lex). These are the legacy lexical front-end.
sema.ResolvePackage / sema.Info already expose the same name-keyed facts
(FuncSignatures with Mode, Structs, Sealed). Migrating removes two of the last
live consumers of analyze/scan, unblocking US-010 (deletion).

## Old code (what exists today)

- typecheck.go: `Package.Tables *analyze.Tables`, built via
  `analyze.BuildPackage(srcs)`. Helpers `goalPosition` / `offsetLineCol`
  (used only by implements.go).
- mustuse.go: reads `p.Tables.FuncSignatures[...]`, compares
  `sig.Mode != analyze.ModeResult`, ranges `p.Tables.Structs`.
- implements.go: `p.Tables.Sealed[iface]`; `implementsClauses` uses
  `scan.Lex` + `textedit.IsIdent` to find `type X struct ... implements ...`
  and splits a comma-separated interface list textually.
- typecheck_test.go: asserts `p.Tables.FuncSignatures["greet"]`.

## New code (goals)

- `Package.Sema *sema.Info`, built via `sema.ResolvePackage(goalFiles)` where
  `goalFiles` are parsed once with the goal `parser.ParseFile` (the same parse
  backend.TranspilePackage already performs, so it cannot fail where transpile
  succeeded). Store the parsed goal files on the Package for the implements walk.
- mustuse.go reads `p.Sema.FuncSignatures` / `p.Sema.Structs` and compares
  against `sema.ModeResult` (already imported).
- implements.go reads `p.Sema.Sealed` and locates `implements` clauses by
  walking the goal AST (`*ast.GenDecl` TYPE -> `*ast.TypeSpec` ->
  `*ast.StructType.Implements`), reading the asserted interface off
  `ImplementsClause.Type` and the clause position off `ImplementsClause.Implements`
  (goal token.Pos already carries 1-based Line/Col, so goalPosition/offsetLineCol
  are dropped).

## What must NOT change (preserved behavior)

- Diagnostic codes, severities, messages, and positions of every depth check
  (CheckImplements, CheckMustUse, CheckNoZero). All existing depth-check tests
  and the corpus conformance tier must stay green.
- Public typecheck API (Load, Check* funcs, Diagnostic, Package methods).

## Note on capability

The goal parser models a single interface per `implements` clause (one
`Type Expr`), so the legacy scan-based comma-list support is dead capability —
goal source cannot express `implements I, J`. The AST walk faithfully covers
everything the parser (and thus the live language) accepts.
