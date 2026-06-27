# Implementation Plan — US-002 interp value model

## File Inventory

### New Files
| File | Purpose |
|------|---------|
| `internal/interp/value.go` | The `Value` type, its `Kind` discriminant, the universal `Variant` tagged union, and the constructors/accessors/Equal/String methods. |
| `internal/interp/value_test.go` | Stdlib `testing` unit tests constructing a Variant + each primitive/composite Value, reading a Variant field by name, and asserting Equal and String(). |

### Modified Files
None. This is a new leaf package with no callers yet.

## Package Structure

```
internal/
  interp/            (new)
    value.go
    value_test.go
```

Sibling to `internal/cap` and `internal/sema`. Dependency-free (stdlib only),
mirroring `internal/cap`'s style: package doc referencing REWRITE-ARCHITECTURE.md.

## Dependency Graph

1. `value.go` — Kind constants, Value struct, Variant struct (no deps).
2. `value_test.go` — depends on 1.

## Interface Contracts

```go
package interp

type Kind int

const (
    KindNil Kind = iota
    KindInt
    KindFloat
    KindString
    KindBool
    KindStruct
    KindSlice
    KindMap
    KindFunc
    KindVariant
)

// Value is the uniform runtime value.
type Value struct {
    Kind Kind
    // payload fields, one used per Kind:
    Int     int64
    Float   float64
    Str     string
    Bool    bool
    Struct  *StructValue   // KindStruct
    Slice   []Value        // KindSlice
    Map     *MapValue      // KindMap
    Func    *FuncValue     // KindFunc
    Variant *Variant       // KindVariant
}

// Variant is the universal tagged union for enum / Result / Option.
type Variant struct {
    TypeID string           // declared type identity, e.g. "Result", "Option", enum name
    Tag    string           // discriminant, e.g. "Ok", "Err", "Some", "None", enum case
    Fields map[string]Value // named payload
}

// Constructors
func IntVal(int64) Value
func FloatVal(float64) Value
func StrVal(string) Value
func BoolVal(bool) Value
func NilVal() Value
func StructVal(typeID string, fields map[string]Value) Value
func SliceVal(elems ...Value) Value
func MapVal(entries map[string]Value) Value   // string-keyed map for v1
func FuncVal(name string) Value               // minimal carrier; callable wiring is US-004+
func VariantVal(typeID, tag string, fields map[string]Value) Value

// Field reads a Variant payload field by name; ok=false when absent or non-variant.
func (v Value) Field(name string) (Value, bool)

func (v Value) Equal(other Value) bool
func (v Value) String() string
```

(StructValue/MapValue/FuncValue are small wrapper structs so structs/maps/funcs
have a distinct identity and ordered/typed representation.)

## Integration Points

None yet. Later stories (US-003 Env, US-004 interpreter entry, US-012+ sum-type
eval) import `internal/interp` and consume `Value`/`Variant`.

## Testing Strategy

`internal/interp/value_test.go`, package `interp`, stdlib `testing` only (NO
testify, per project constraint). Tests:
- Construct each primitive + composite Value and a Variant.
- `Field("x")` reads a Variant payload back by name; missing field returns ok=false.
- `Equal` true for equal values (incl. Variants), false for differing tag/type/fields.
- `String()` is non-empty and readable for every Value kind.
