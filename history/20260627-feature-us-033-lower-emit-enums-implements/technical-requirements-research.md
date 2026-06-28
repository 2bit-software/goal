# Technical Requirements / Research — US-033

## Where the work lands

- `internal/backend/emit.go` — the AST Go emitter. Add goal-construct lowering
  for EnumDecl, SealedInterfaceDecl, struct `implements`, VariantLit, and
  data-less variant SelectorExpr, consulting `*sema.Info`.
- `internal/backend/backend.go` — `Transpile` must call `sema.Resolve(file)`
  instead of `sema.New()` so the emitter has the enum/sealed/methods facts; and
  `goBackend.Emit` must thread `info` into the emitter.

## Reference (legacy splice encoding — known-good)

- `internal/pass/enums.go`: `genEnum` (interface + per-variant structs + marker
  methods), `genInterface` (`type X interface{ isX() }`), `construct`
  (`Enum(Enum_V{Field: x})`), `exported` (capitalize first rune).
- `internal/pass/implements.go`: sealed -> `genMarker` (`func (T) isI() {}`);
  ordinary -> `var _ I = T{}` or `(*T)(nil)` when T has a pointer-receiver
  method (`scanPointerReceivers`).

## sema facts consumed

- `info.Enums[name]` (*Enum: Variants []Variant{Name, Fields []Field{Name,Type}},
  VSet) — drives genEnum + construction membership.
- `info.Sealed[name]` (bool) — selects marker-method vs var-assertion for
  implements.
- Pointer-receiver detection: scan the *ast.File for FuncDecl with a pointer
  receiver (Recv field Type is *ast.StarExpr) — sema.Methods is star-stripped so
  it cannot answer this; compute a local set in the emitter.

## Key AST facts

- Data-less `Status.Pending` parses to *ast.SelectorExpr (X enum Ident, Sel
  variant) — NOT a VariantLit. Lower in the SelectorExpr case, guarded by
  info.Enums membership so ordinary/qualified selectors (io.Writer) are untouched.
- Payload `Status.Active(since: now())` parses to *ast.VariantLit with
  *ast.LabeledArg args. Values emitted recursively so nested constructions lower
  for free.
- Struct `implements` lives on StructType.Implements (*ast.ImplementsClause,
  Type an *Ident or *SelectorExpr). The marker/assertion is a SEPARATE top-level
  decl emitted after the type decl.

## Verification

- prd verifyCommands: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
- AC2: a backend test runs the 4 01-enums + 3 07-implements transpile cases
  through `backend.Transpile` + `corpus.RunCompile` (behavioral tier).
