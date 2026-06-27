# Technical Requirements / Research — US-049

## Build on the existing AST symbol graph

US-048 (`internal/lsp/definition.go`) already parses the open buffer
(`parser.ParseFile`), builds a name-keyed declaration index over `file.Decls`,
and walks references keyed by structural parent (`refVisitor`). Hover reuses the
same shape: resolve the symbol under the cursor, then render its declaration's
signature + doc instead of its declaration range.

## Signature rendering

A `FuncDecl` carries `Type *ast.FuncType` (params/results) and an optional
`Doc *ast.DocComment`. The faithful signature is the raw source slice from the
declaration start (`FuncDecl.Pos().Offset`, which already accounts for the
`from`/`derive` modifier and the `func` keyword) through `Type.End().Offset`,
whitespace-collapsed — this includes the function name and a Result/Option
result type verbatim. Enum/sealed/type/variant symbols render a short header.

## Protocol

Add `HoverParams` (document + position, same as `DefinitionParams`), `Hover`
(`contents: MarkupContent`), and `MarkupContent` (`kind`, `value`). Advertise
`hoverProvider: true` and route `textDocument/hover`. A nil `*Hover` marshals to
JSON null, matching the definition/semantic-token best-effort contract.

## Test

Mirror `definition_test.go`: a sample with a Result-returning function; assert
the hover string contains the function's signature, plus null fallbacks for
no-symbol / unparseable / unknown URI, and a server capability assertion.
