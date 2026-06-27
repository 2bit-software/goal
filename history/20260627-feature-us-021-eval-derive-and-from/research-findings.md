# Research Findings — US-021

This is an internal-codebase feature (no external/web research required). Findings
are drawn from reading the existing implementations.

## How `from func` works at runtime (already wired)
- `from func name(...) T { ... }` parses to ast.FuncDecl with Mod == FuncFrom and
  a body. `registerFuncs` skips only `fn.Body == nil` / receivers; a from-func has
  a body, so it is registered as an ordinary callable in the root scope.
- sema.Resolve records the conversion in `Info.FromRegistry[[2]string{src,tgt}] =
  ConvEntry{Name, Fallible}` (it is NOT in FuncSignatures for the derive purpose).
- The closed-E `?` path already calls a from-func by name through `callConversion`
  -> root.Lookup(name) -> callFunc. US-021 reuses exactly this invocation pattern.

## How `derive func` is lowered by the Go backend (the strategy to mirror)
- emit.go `deriveDecl` reads srcName/srcType from the first param, target type +
  fallibility from `deriveTarget(results)`, and overrides from `deriveOverrides`.
- `genConversion` declares `var out T`, emits overrides, then for each
  un-overridden target field calls `resolveField(dst, srcExpr, sf, tf, fallible)`.
- `resolveField` chooses by (sf -> tf): identity, registry total, registry
  fallible (threads `err`), pointer/Option recursion, slice/array/map recursion,
  nested in-file struct recursion (`deriveBody`). Unresolvable => located fail.
- Pure string splitters in lower.go (typeExprString, derefType, findSemaField,
  ptrInner, arrElem, mapKV, elemConv, deriveTarget) operate on resolved sema type
  strings. The interpreter cannot import backend (US-022), so the needed splitters
  are mirrored locally in interp.

## Interpreter gap
- A derive func is bodyless (or has a `_`/`...derive` body the ordinary evaluator
  can't run), so it is never a normal callable. US-021 registers derive decls into
  a dedicated map and intercepts a call to one, producing the target struct Value
  field-by-field — the runtime analogue of genConversion.

## Decision
Implement the strategies the file-mode features/12 fixtures exercise: identity,
registry total, registry fallible, nested in-file struct recursion, slice/array
recursion, map recursion, plus bodied overrides (`Field: expr`, `Field: _`,
`...derive(src)` fill). Pointer/Option recursion is refused loudly (the interp
models no address-of); this is honest deferral, never a silent zero.

**Confidence: High** — mirrors a known-good backend implementation over the same
sema facts.
