# Technical Plan — US-038 Lower and emit defaults and assert

## Files

### `internal/backend/lower.go` (edit — add encoders/facts)
- `typeDeclMap` analogue is built as an emitter method (see emit.go) since it
  renders alias underlying types via the emitter; lower.go holds the pure helpers:
- `zeroLit(typ string, decls map[string]string, depth int) string` — mirrors
  `analyze.ZeroLit`: pointer/slice/map/chan/func/interface/any/error -> `nil`;
  array `[N]T` -> `T{}`; string -> `""`; bool -> `false`; numerics -> `0`; a named
  type resolved through `decls` (struct -> `T{}`, interface -> `nil`, else recurse);
  unknown named -> `T{}`. depth guards alias chains.
- `zeroSafety(typ string, decls map[string]string, info *sema.Info, depth int) string`
  — mirrors `internal/pass.zeroSafety`: returns "" when the zero is safe, else a
  reason string. nil pointer/map/chan/func/method-iface -> reason; bare
  `interface{}`/`any`/`error`/slice/array/primitive -> safe; a sum type
  (`info.Enums[base]` or `info.Sealed[base]`) -> reason; named alias chain via
  `decls`; unknown named -> safe.
- `baseType(t string) string` — local mirror of `scan.BaseType` (strip `*` and
  `pkg.` qualifier) so lower.go stays free of an `internal/scan` import.

### `internal/backend/emit.go` (edit — dispatch + file-level injection)
- `emitter` gains `typeDecls map[string]string`.
- `emitFile`: after constructing `e`, set `e.typeDecls = e.buildTypeDecls(f)`.
- `buildTypeDecls(f *ast.File) map[string]string` (emitter method): walk `f.Decls`
  for `*ast.GenDecl` with `Tok.String()=="type"`; per `*ast.TypeSpec`:
  StructType -> "struct", InterfaceType -> "interface", else -> `e.exprText(type)`.
- `exprText(x ast.Expr) string` (emitter method): render x to text by swapping a
  temporary `strings.Builder` into `e.b`, calling `e.expr(x)`, capturing, restoring.
- `file()`: after the package clause and before the decls loop, when
  `needsFmtImport(f)` and not `importsPkg(f,"fmt")`, emit `import "fmt"\n\n`.
- `stmt()`: add `case *ast.AssertStmt: e.assertStmt(s)`.
- `assertStmt(s *ast.AssertStmt)`: emit `if !(<cond>) { panic(<msg>) }`. Bare form
  panics `strconv.Quote("assertion failed: "+condText)`; message form panics
  `Quote("assertion failed: "+condText+": ") + " + fmt.Sprintf(" + <Msg>[, <Args>] + ")"`.
  `condText` from `e.exprText(s.Cond)`; the real `cond` re-emitted via `e.expr`.
- CompositeLit emission routed through a new `compositeLit(x *ast.CompositeLit)`:
  emits `Type{` then each elt comma-separated; a `*ast.SpreadElement` whose X is
  `Ident "defaults"` expands via `defaultEntries`; any other spread -> `e.fail`
  (derive is US-039).
- `defaultEntries(x *ast.CompositeLit) []string`: require `x.Type` an `*ast.Ident`
  naming a known struct (`e.info.Structs`); compute present field names from the
  literal's `KeyValueExpr` keys; for each absent field run `zeroSafety` (fail on
  unsafe) and append `name: zeroLit(type)`.

### `internal/backend/lower.go` helpers (cont.)
- `needsFmtImport(f *ast.File) bool` — Walk for `*ast.AssertStmt` with `Msg != nil`.
- `importsPkg(f *ast.File, path string) bool` — scan import GenDecls / ImportSpecs
  for `"path"`.
- `presentFieldNames(elts []ast.Expr) map[string]bool` — collect KeyValueExpr keys.
- `structFieldsOf(info *sema.Info, name string) ([]sema.Field, bool)` — nil-safe
  `info.Structs[name]` lookup.

## Tests — `internal/backend/backend_test.go` (edit)
- `defaultsAssertCases` = the 3 `features/08-no-zero-value/examples/*.goal` + 3
  `features/10-assert/examples/*.goal` inputs.
- `TestASTEngineDefaultsAssertBehavioralTier` — run each through
  `corpus.RunCompile(repoRoot, c, corpus.TranspilerFunc(backend.Transpile))`
  (build+vet), `-short`-skipped (spawns the go toolchain), loud zero-case guard.
- `TestASTEngineDefaultsAssertEncoding` — pin shapes: expanded `...defaults`
  yields explicit `email: "", active: false, logins: 0`; bare assert ->
  `if !(amount > 0) { panic("assertion failed: amount > 0") }`; message assert ->
  the `fmt.Sprintf` form + injected `import "fmt"`.

## Dependency order
1. lower.go helpers (`zeroLit`, `zeroSafety`, `baseType`, `needsFmtImport`,
   `importsPkg`, `presentFieldNames`, `structFieldsOf`).
2. emit.go dispatch (`exprText`, `buildTypeDecls`, `assertStmt`, `compositeLit`,
   `defaultEntries`, file() fmt injection, stmt() AssertStmt case).
3. Tests.

## Out of scope
`...derive(s)` spread and `derive func` lowering (US-039); exact golden parity
(US-042). Behavioral tier (build+vet) is the gate here, not byte-exact goldens.
