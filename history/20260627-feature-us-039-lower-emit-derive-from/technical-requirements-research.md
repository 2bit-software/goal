# Technical Requirements / Research — US-039

## Where the work lands

- `internal/backend/emit.go` — `funcDecl` currently fails on `ast.FuncDerive`.
  Add a `deriveDecl` path that emits the generated conversion function. Extend
  `compositeLit` to recognize the `...derive(src)` spread inside a derive body
  (today only `...defaults` is handled; `...derive` is the other SpreadElement).
- `internal/backend/lower.go` — port the known-good conversion encoders from
  `internal/pass/derive.go`, but read facts off `sema.Info`:
  - `Info.Structs[name] []sema.Field{Name,Type}` for source/target fields.
  - `Info.FromRegistry map[[2]string]sema.ConvEntry{Name,Fallible}` for leaf
    conversions, keyed by sema `typeString` output.
  - Type strings rendered to match sema's `typeString` (the registry/struct keys).

## Lowering shapes (mirror internal/pass/derive.go, known-good build+vet-clean)

- identity: `out.F = src.F`
- total leaf: `out.F = conv(src.F)`
- fallible leaf: `v, err := conv(src.F); if err != nil { return out, err }; out.F = v`
- slice `[]A->[]B`: `make` + indexed `for i := range`
- array `[N]A->[N]B`: in-place `for i := range`
- map `map[K]A->map[K]B`: `make` + `for k,v := range`
- pointer/Option `*A->*B` / `Option[A]->Option[B]`: nil-guarded re-address
- nested struct `A->B`: temp var + field-by-field body

## Gensym

Temp names use the emitter's scope-aware `gensym` (no `__goal_` prefix), so the
output is build+vet-clean but not byte-identical to the splice goldens (exact
golden parity is US-042).

## Test

- `TestASTEngineDeriveBehavioralTier` — run features/12 slice/from_storage/
  to_storage through `backend.Transpile` + `corpus.RunCompile` (build+vet),
  `-short`-skipped.
- `TestASTEngineDeriveEncoding` — pin the identity / total-leaf / fallible-leaf /
  slice-recursion shapes and the `from func` strip.

## Out of scope

- The foreign-derive package fixture (US-009) runs through the splice engine via
  corpus.RunPackage; this story only covers the three file-mode features/12 cases.
- Exact golden regeneration (US-042).
