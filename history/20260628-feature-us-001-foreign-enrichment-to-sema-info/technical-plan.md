# Technical Plan — US-001

## Files

1. `internal/sema/sema.go`
   - Add field `ForeignMethods map[string]FuncSig` to `Info`, documented as
     `pkg.Type.Method -> FuncSig` for imported method signatures.

2. `internal/sema/resolve.go`
   - Initialize `ForeignMethods: map[string]FuncSig{}` in `Resolve`.
   - Add `maps.Copy(info.ForeignMethods, o.ForeignMethods)` to `Merge`.

3. `internal/sema/foreign.go` (new)
   - `type DirResolver func(importPath, fromDir string) (string, error)`
   - `func DefaultResolver(importPath, fromDir string) (string, error)` plus
     helpers `moduleResolve`, `readModulePath`, `goListResolve`, `isDir`,
     `lastSegment` (ported from analyze/foreign.go).
   - `func EnrichForeign(info *Info, imports []*ast.ImportSpec, dir string, resolve DirResolver) []error`
     - resolve defaults to DefaultResolver when nil.
     - iterate `imports`; for each, read alias from `imp.Name` (nil/`_`/`.`
       handled), path by unquoting `imp.Path.Value`; skip `_`/`.` blanks where
       no qualifier applies; dedup by path.
     - resolve dir, call `foreignDecls`, merge structs into `info.Structs`,
       methods into `info.ForeignMethods`, receiver-less funcs into
       `info.FuncSignatures`.
   - `foreignDecls` + helpers (`exportedFields`, `goTypeString`, `isGoBuiltin`,
     `foreignRecvBase`, `endsInErrorAST`, `resultArityGo`, `exprText`) ported
     from analyze, producing sema.Field / sema.FuncSig.
   - Uses stdlib go/ast + go/parser to read foreign Go; NO scan.Lex, NO
     analyze.ParseImports.

4. `internal/sema/foreign_test.go` (new)
   - Parse a goal source (parser.ParseFile) importing the shared
     `../analyze/testdata/extpkg` fixture via a `derive func`.
   - Inject a fixture resolver mapping the import path to the on-disk dir.
   - Call sema.EnrichForeign(info, file.Imports, ".", resolve) and
     analyze.EnrichForeign(tables, []string{src}, ".", resolve).
   - Assert Structs + ForeignMethods entries match field-for-field.
   - Assert no scan.Lex import in the new file (dependency gate via go list, or
     a source-grep test).

## Verification

- `task check`, `task build`, plus `go test ./internal/sema/...`.
