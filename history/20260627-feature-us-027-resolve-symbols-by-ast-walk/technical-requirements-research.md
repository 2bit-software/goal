# Technical Requirements / Research — US-027

## Existing surface

- `internal/analyze` (analyze.go, methods.go, foreign.go) builds `*Tables`
  keyed by symbol name from a flat `scan.Lex` token stream. Known structural
  weakness: `parseStructBody` splits struct field text on whitespace and treats
  the last token as the type, so any type containing a top-level comma (a
  func-typed field, a multi-arg generic without bracket awareness) is split
  wrong. US-027 fixes this by reading fields off the parsed AST.
- `internal/ast` already models every node sema needs: `EnumDecl`/`Variant`/
  `PayloadField`, `StructType`/`FieldList`/`Field`, `FuncDecl` (with `Mod`
  FuncFrom/FuncDerive, `Recv`, `Type *FuncType`), `SealedInterfaceDecl`,
  `IndexExpr`/`IndexListExpr` for `Result[T,E]` / `Option[T]`.
- `internal/parser` `ParseFile(src)` yields the `*ast.File`.

## Approach

- Add fields + supporting types to `sema.Info` mirroring the analyze facts
  (FuncSig/Enum/Variant/Field/ConvEntry/Method + a Mode enum with the SAME
  iota order as analyze so int values compare equal).
- Add `sema.Resolve(*ast.File) *Info` that walks `File.Decls`:
  - `*ast.EnumDecl` -> Enums (+ VSet/FieldSet)
  - `*ast.GenDecl` Tok=TYPE with `*ast.StructType` -> Structs (per name, type via
    a compact `typeString` AST printer)
  - `*ast.SealedInterfaceDecl` -> Sealed
  - `*ast.FuncDecl`: plain -> FuncSignatures (Result/Option detection off the
    result FieldList's IndexExpr/IndexListExpr); FuncFrom/FuncDerive -> FromRegistry
    (src = first param type, target/fallible from results); methods (Recv != nil)
    -> Methods keyed by receiver type (star-stripped).
- `typeString` is a small position-free AST type printer (Ident/Selector/Star/
  Array/Map/Index(+List)/Func/Chan/Ellipsis/Paren) producing source-equivalent
  text. The parity test normalizes whitespace before comparing type strings, so
  it asserts semantic equality rather than exact spelling.
- `sema` must NOT import `analyze` (it is the replacement). The parity test lives
  in `package sema` and imports `parser` + `analyze` (neither imports sema, so no
  cycle).

## Verify

- prd.json verifyCommands: `go build ./...`, `go vet ./...`,
  `go test ./... -count=1`.
- Story AC: the new `internal/sema` parity test, plus the embedded-comma struct
  field case.
