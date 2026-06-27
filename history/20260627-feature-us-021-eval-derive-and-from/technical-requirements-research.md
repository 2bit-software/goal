# Technical Requirements / Research — US-021

## Existing seams to reuse

- `from func` declarations already work: they carry a body and are registered as
  ordinary callables by `registerFuncs`. Their src->tgt mapping lives in
  `sema.Info.FromRegistry[[2]string{src, tgt}] -> ConvEntry{Name, Fallible}`.
  A registry conversion is invoked by calling the registered function by name
  via `callFunc` (the pattern `callConversion` already uses for closed-E `?`).

- `derive func` declarations (ast.FuncDecl with Mod == ast.FuncDerive) are NOT
  ordinary callables: a bodyless derive has no body, and a bodied derive's body
  contains `_` skip values and a `...derive(src)` spread that the ordinary
  composite-literal evaluator cannot run. They must be intercepted and evaluated
  specially.

## Backend reference (internal/backend/emit.go + lower.go)

The Go backend lowers a derive in `deriveDecl` -> `genConversion` -> `resolveField`
/ `deriveBody`, using pure string-level type splitters in lower.go
(typeExprString, derefType, findSemaField, ptrInner, arrElem, mapKV, elemConv,
deriveTarget). The interpreter mirrors the *strategy selection* but produces a
runtime Value instead of Go text.

Field conversion strategies (by source sema type sf -> target sema type tf):
- identity (sf == tf): copy the value through.
- registry total: `reg[{sf,tf}]` non-fallible -> call conv(value).
- registry fallible: call conv(value) -> (val, err); a non-nil err short-circuits
  the derive (it returns (target, err)).
- nested in-file struct recursion: both sf and tf are in-file structs -> recurse
  field-by-field.
- slice/array recursion: `[]A`/`[N]A` -> `[]B`/`[N]B` with a total element conv.
- map recursion: `map[K]A` -> `map[K]B` with a total element conv.

## Plan

- Add `internal/interp/derive.go` with the value-producing conversion logic and
  the type-string splitters local to interp (interp must not import backend).
- Register derive decls into a new `ip.derives` map (skip them in registerFuncs).
- Intercept a call whose callee Ident names a derive in `evalCallMulti`.

## Test fixture

`derive_nested_struct.goal` exercises identity (Name), a registry bridge
(parseCode: string -> Code for the Zip field) and nested struct recursion
(Addr -> AddrV2). It is the canonical 12-derive-convert shape for the unit test.
