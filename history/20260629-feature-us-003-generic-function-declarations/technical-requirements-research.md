# Technical Requirements / Research

Localized ~3-spot fix per prd notes:

- `parser.go` `parseFuncDecl` (~line 347) must call the existing
  `atTypeParams`/`parseTypeParams` helpers (~lines 281, 296, 307) when a
  `[` follows the function name.
- Add a `TypeParams` field to `ast.FuncType` (currently only `TypeSpec`
  carries one, ast.go ~line 261).
- Emit the type-parameter list in the backend `funcDecl`/`funcSig`
  (backend/emit.go ~lines 360, 439).

Methods (receiver present) must NOT gain type params on the func itself —
generic type params belong to top-level generic funcs.
