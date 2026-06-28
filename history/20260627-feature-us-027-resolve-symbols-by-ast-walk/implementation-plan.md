# Implementation Plan — US-027 Resolve symbols by AST walk

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/sema/resolve.go` | `Resolve(*ast.File) *Info` plus the AST walk that populates each fact table, and the `typeString` AST type printer. |
| `internal/sema/resolve_test.go` | Parity test vs `analyze.Build` over representative inputs + the embedded-comma struct-field case. |

### Modified Files
| File | Changes |
|------|---------|
| `internal/sema/sema.go` | Expand `Info` from a placeholder to carry the name-keyed fact maps; add the supporting types (`Mode`, `FuncSig`, `Field`, `Variant`, `Enum`, `ConvEntry`, `Method`). `New()` keeps returning an empty (nil-map) `Info` for the plain-Go back-end path. |

No back-end / driver changes: `internal/backend` continues to call `sema.New()`
(US-032+ wires `Resolve` into emit). This keeps blast radius minimal.

## Package Structure

```
internal/sema/
  sema.go         (Info + fact types; New)        [modified]
  resolve.go      (Resolve + walk + typeString)   [new]
  sema_test.go    (existing TestNewReturnsInfo)    [unchanged]
  resolve_test.go (parity + comma-bug test)        [new]
```

## Dependency Graph

1. `sema.go` types (`Info` fields + `Mode`/`FuncSig`/`Field`/`Variant`/`Enum`/
   `ConvEntry`/`Method`) — depend only on nothing (plain data).
2. `resolve.go` `typeString` — depends on `internal/ast` node set only.
3. `resolve.go` `Resolve` + per-decl walkers — depend on 1 and 2.
4. `resolve_test.go` — depends on `sema.Resolve`, `parser.ParseFile`,
   `analyze.Build`.

## Interface Contracts

```go
// sema.go
type Mode int
const ( ModeNone Mode = iota; ModeResult; ModeResultClosed; ModeOption ) // SAME order as analyze

type FuncSig struct { Name string; Mode Mode; T, E string; Arity int; EndsInError bool }
type Field   struct { Name, Type string }
type Variant struct { Name string; Fields []Field }
type Enum    struct { Name string; Variants []Variant; VSet map[string]bool; FieldSet map[string]map[string]bool }
type ConvEntry struct { Name string; Fallible bool }
type Method  struct { Name, Sig, Raw string; Arity int; EndsInError bool }

type Info struct {
    FuncSignatures map[string]FuncSig
    Enums          map[string]*Enum
    Sealed         map[string]bool
    Structs        map[string][]Field
    FromRegistry   map[[2]string]ConvEntry
    Methods        map[string][]Method
}
func New() *Info                  // empty placeholder (nil maps) — back-end seam
func Resolve(f *ast.File) *Info   // fully-initialized, populated by AST walk

// resolve.go
func typeString(x ast.Expr) string
```

## Integration Points

- `internal/backend/backend.go::Transpile` already calls `sema.New()`. Unchanged
  this story (Emit does not read Info). `Resolve` is the future entry point.
- Parity reference: `internal/analyze/analyze.go::Build` — the test asserts
  resolved-fact equality for representative symbols.

## Testing Strategy

- `internal/sema/resolve_test.go` (`package sema`, stdlib `testing` only):
  - `TestResolveMatchesAnalyze`: parse a representative goal source (enum w/
    payload, struct, plain func, Result/Option/closed-Result funcs, `from func`,
    `derive func`, value+pointer methods, sealed interface); build via both
    `analyze.Build` and `sema.Resolve(parser.ParseFile(...))`; assert each named
    symbol's facts agree (type strings compared whitespace-insensitively).
  - `TestResolveStructCommaFieldType`: a struct with a func-typed field
    `cb func(int, string)`; assert sema resolves exactly the expected fields with
    complete types (the analyze comma-split bug case).
- No external test package needed (`parser`/`analyze` do not import `sema`).
