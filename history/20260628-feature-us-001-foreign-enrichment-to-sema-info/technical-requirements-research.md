# Technical Requirements / Research — US-001

## Signature

```go
func EnrichForeign(info *Info, imports []*ast.ImportSpec, dir string, resolve DirResolver) []error
```

`ast` here is goal's internal/ast. `*ast.ImportSpec` carries `Name *Ident`
(alias, may be `_`/`.`/nil) and `Path *BasicLit` (quoted import path).

## Reuse

- The Go-toolchain reader logic (resolve import path -> directory -> parse
  exported decls) is ported from internal/analyze/foreign.go. The foreign Go
  package is read with the stdlib go/ast + go/parser, exactly as analyze does,
  so field/method rendering matches analyze byte-for-byte and the parity test
  holds.
- Struct field types are rendered qualified by the import alias (analyze's
  goTypeString), NOT sema.typeString (which does not qualify) — foreign fields
  must read as `*alias.Type` to align with the goal source spelling.
- Methods are keyed `alias.RecvBase.Method` and valued with a sema.FuncSig
  carrying only the `?`-relevant Arity + EndsInError (Mode left ModeNone),
  mirroring analyze's foreign method entries.

## Differences from analyze.EnrichForeign

- Imports come from the parsed AST (file.Imports), not from re-lexing source via
  scan.Lex / analyze.ParseImports.
- This driver loads every import in the supplied list (the AST already scopes
  them to the file). The "needed-alias" pre-filtering analyze does (to avoid
  parsing unrelated deps) is a separate optimization concern handled by the
  caller's import list.

## Resolver

DirResolver / DefaultResolver are ported from analyze (same-module walk to
go.mod, then `go list` fallback). The DefaultResolver is offline for the common
in-module case.
