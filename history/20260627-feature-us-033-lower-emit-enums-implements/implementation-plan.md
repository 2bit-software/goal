# Implementation Plan — US-033 Lower and emit enums and implements

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/backend/lower.go` | Goal-construct emission helpers consumed by emit.go: enum/sealed encoding (genEnum/genInterface), implements marker/assertion, exported(), pointer-receiver scan. Keeps emit.go's plain-Go core readable. |
| `internal/backend/testdata/enums_implements_test.goal` (optional) | Not needed — the 7 corpus example cases already exist; the test points at them directly. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/backend/backend.go` | `Transpile` resolves real semantics: `sema.Resolve(file)` instead of `sema.New()`. `goBackend.Emit` threads `info` into `emitFile`. |
| `internal/backend/emit.go` | `emitter` gains `info *sema.Info` + `pointerRecv map[string]bool`. `emitFile(f, info)`. `decl` handles `*ast.EnumDecl`/`*ast.SealedInterfaceDecl`. `structType` drops the `implements` clause (no longer fails). After a struct TypeSpec carrying `implements`, emit the marker/assertion decl. `expr` lowers `*ast.VariantLit` and data-less enum `*ast.SelectorExpr` to the construction encoding. |
| `internal/backend/backend_test.go` | Add `TestASTEngineEnumsImplementsBehavioralTier` driving the 4 `features/01-enums` + 3 `features/07-implements` transpile cases through `backend.Transpile` + `corpus.RunCompile`; plus a focused transpile+format assertion that the enum/implements encoding text appears. |

## Encoding (mirrors internal/pass, known-good)

- Enum `Name`:
  - `type Name interface{ isName() }`
  - per variant: `type Name_V struct{}` (data-less) or `type Name_V struct { Exported Type ... }`
  - per variant: `func (Name_V) isName() {}`
- Construction:
  - data-less SelectorExpr `Name.V` -> `Name(Name_V{})`
  - payload VariantLit `Name.V(label: x)` -> `Name(Name_V{Label: x})` (labels exported, values recursive)
- Sealed `interface Name {}` -> `type Name interface{ isName() }`
- Struct `T implements I`:
  - drop clause, emit plain struct
  - `info.Sealed[I]` -> `func (T) isI() {}`
  - else pointer-recv(T) -> `var _ I = (*T)(nil)`
  - else -> `var _ I = T{}`

## Lowering placement decision

Fold lowering into the emitter (no separate `lower` package): the arch doc marks
`lower`/`ir` optional, and emit.go already does direct AST->Go-text emission.
The "lower" naming is satisfied by `lower.go` holding the goal-construct encoders.

## Disambiguation guards

- A `*ast.SelectorExpr` is lowered as a data-less variant ONLY when `X` is an
  `*ast.Ident` present in `info.Enums` and `Sel` is in that enum's `VSet`.
  Ordinary/qualified selectors (e.g. `io.Writer`, `c.n`) are untouched.
- A `*ast.VariantLit` is lowered when its `Enum` is an `*ast.Ident` in
  `info.Enums`; otherwise it is an unsupported-node error (none occur in scope).

## Test Strategy

- prd verifyCommands: `go build ./...`, `go vet ./...`, `go test ./... -count=1`.
- Behavioral tier (AC2): backend test over the 7 example cases via
  `corpus.RunCompile` (build + vet in a temp module), `-short`-skipped (spawns
  the go toolchain), loud zero-case guard.
- Focused unit: transpile `features/01-enums/examples/traffic.goal` and assert
  the output contains `type Light interface{ isLight() }` and
  `Light(Light_Red{})`; transpile a sealed + a pointer-recv case and assert the
  marker / `(*T)(nil)` forms.

## Risks

- Switching `Transpile` to `sema.Resolve` must stay inert for plain-Go files —
  Resolve returns initialized empty maps, so the existing plain-Go tests
  (US-026/US-032) keep passing. Verified by running the full suite.
