# Scope — US-008 Migrate project and sourcemap off scan

## What is being refactored and why

Two packages still depend on the token scanner (`internal/scan`), the last
lexer crutch before self-host. They must derive their facts from the goal AST
(`internal/parser` + `internal/ast`) instead.

## Old code

- `internal/project/project.go`
  - `PackageClause(src)` uses `scan.Lex` to find a `package <name>` clause.
  - Imports `goal/internal/scan` and `goal/internal/textedit` (only for
    `textedit.IsIdent`, used inside `PackageClause`).
- `internal/pipeline/sourcemap.go`
  - `declSites(src)` uses `scan.Lex` to find top-level decl keywords at line
    start (depth-0), filtering with `textedit.IsLineStart`.
  - `declName(toks, i)` uses `scan.MatchParen` to skip a method receiver.
  - `isDeclKeyword` enumerates func/type/var/const/import/enum.
  - Imports `goal/internal/scan` and `goal/internal/textedit`.

## New code (goals / constraints)

- `PackageClause` parses with `parser.ParseFile` and returns `file.Name.Name`,
  guarded by verifying the source at `file.Package.Offset` actually begins with
  the `package` keyword (the goal parser's `expect` synthesizes a name token even
  when the clause is missing, so `func main(){}` must still yield ""). Tolerant:
  reads the name even on later body parse errors, like the old tolerant lexer.
- `declSites` parses with `parser.ParseFile` and walks `file.Decls`, using
  `decl.Pos().Offset` (always the keyword offset / line start in gofmt'd Go —
  confirmed `FuncType.Func` is set even for methods, parser.go:350) and an
  AST-based `declName(ast.Decl)`.
- Neither package imports `internal/scan` or `internal/analyze` afterward.

## What must NOT change

- `PackageClause` public signature and semantics (clause in strings/comments
  ignored; "" when absent). Existing project tests must pass.
- `AddLineDirectives` output for the existing source-map test (func mapped to
  its source line; synthesized decls identity-anchored).
- No golden/conformance test asserts `//line` directive text; conformance is
  behavioral (compiles + runs), so decl-naming nuances don't affect runtime.

## Notes / minor divergences

The legacy lexer skipped `sealed`/`from`/`derive` declarations as map sites
(their leading token isn't a decl keyword); the AST walk maps those named decls
to their source lines too. This is strictly more correct for error reporting and
no test asserts the old behavior.
