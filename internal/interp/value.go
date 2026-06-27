// Package interp defines the goscript tree-walking interpreter's runtime model.
//
// This file establishes the uniform runtime VALUE representation. goscript
// interprets the shared AST + sema front-end directly, so every runtime
// construct shares ONE encoding rather than each inventing its own.
//
// Crucially, the interpreter does NOT reuse the Go backend's optimizations
// (REWRITE-ARCHITECTURE.md §4): the Go codegen lowers Result to (T, error) and
// Option to *T, but the interpreter encodes Result, Option, and every enum
// uniformly as the universal tagged union Variant{TypeID, Tag, Fields}. The Go
// backend's lowering is deliberately not imported here.
package interp

import (
	"sort"
	"strconv"
	"strings"

	"goal/internal/ast"
)

// Kind discriminates the dynamic kind of a Value.
type Kind int

// The runtime value kinds. Order is stable.
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

// String returns the readable name of the kind.
func (k Kind) String() string {
	switch k {
	case KindNil:
		return "nil"
	case KindInt:
		return "int"
	case KindFloat:
		return "float"
	case KindString:
		return "string"
	case KindBool:
		return "bool"
	case KindStruct:
		return "struct"
	case KindSlice:
		return "slice"
	case KindMap:
		return "map"
	case KindFunc:
		return "func"
	case KindVariant:
		return "variant"
	default:
		return "Kind(" + strconv.Itoa(int(k)) + ")"
	}
}

// Value is the uniform runtime value. Exactly one payload field is meaningful,
// selected by Kind; the rest are zero.
type Value struct {
	Kind Kind

	Int     int64
	Float   float64
	Str     string
	Bool    bool
	Struct  *StructValue // KindStruct
	Slice   []Value      // KindSlice
	Map     *MapValue    // KindMap
	Func    *FuncValue   // KindFunc
	Variant *Variant     // KindVariant
}

// StructValue is an ordinary struct instance: a named type with named fields.
type StructValue struct {
	TypeID string
	Fields map[string]Value
}

// MapValue is a runtime map. v1 keys are strings (the encoding later eval
// stories build on); the wrapper gives maps a distinct identity from structs.
type MapValue struct {
	Entries map[string]Value
}

// FuncValue is the runtime representation of a function value. It carries the
// declaration it closes over (Decl) and the lexical scope in which it was
// defined (Env) so a call can bind parameters in a fresh child of that scope and
// run the body. Name is retained for rendering. A name-only carrier (Decl == nil)
// is still valid for tests/diagnostics that do not call the function.
type FuncValue struct {
	Name string
	Decl *ast.FuncDecl // the function declaration; nil for a name-only carrier
	Env  *Env          // the lexical scope the function closes over; nil for a name-only carrier
}

// Variant is the UNIVERSAL tagged union. It backs every sum type uniformly:
//   - enums:  TypeID = enum name,   Tag = case name
//   - Result: TypeID = "Result",    Tag = "Ok" | "Err"
//   - Option: TypeID = "Option",    Tag = "Some" | "None"
//
// Fields holds the named payload (empty for data-less variants).
type Variant struct {
	TypeID string
	Tag    string
	Fields map[string]Value
}

// IntVal constructs an int value.
func IntVal(n int64) Value { return Value{Kind: KindInt, Int: n} }

// FloatVal constructs a float value.
func FloatVal(f float64) Value { return Value{Kind: KindFloat, Float: f} }

// StrVal constructs a string value.
func StrVal(s string) Value { return Value{Kind: KindString, Str: s} }

// BoolVal constructs a bool value.
func BoolVal(b bool) Value { return Value{Kind: KindBool, Bool: b} }

// NilVal constructs the nil value.
func NilVal() Value { return Value{Kind: KindNil} }

// StructVal constructs a struct value with the given type identity and fields.
func StructVal(typeID string, fields map[string]Value) Value {
	if fields == nil {
		fields = map[string]Value{}
	}
	return Value{Kind: KindStruct, Struct: &StructValue{TypeID: typeID, Fields: fields}}
}

// SliceVal constructs a slice value from the given elements.
func SliceVal(elems ...Value) Value {
	if elems == nil {
		elems = []Value{}
	}
	return Value{Kind: KindSlice, Slice: elems}
}

// MapVal constructs a map value from the given string-keyed entries.
func MapVal(entries map[string]Value) Value {
	if entries == nil {
		entries = map[string]Value{}
	}
	return Value{Kind: KindMap, Map: &MapValue{Entries: entries}}
}

// FuncVal constructs a name-only function value (no declaration). It is the
// minimal carrier used where the function is not called (rendering, identity
// equality); use FuncDeclVal to construct a callable function value.
func FuncVal(name string) Value {
	return Value{Kind: KindFunc, Func: &FuncValue{Name: name}}
}

// FuncDeclVal constructs a callable function value over the given declaration and
// its defining scope. A call binds parameters in a fresh child of env and runs
// decl.Body.
func FuncDeclVal(decl *ast.FuncDecl, env *Env) Value {
	name := ""
	if decl != nil && decl.Name != nil {
		name = decl.Name.Name
	}
	return Value{Kind: KindFunc, Func: &FuncValue{Name: name, Decl: decl, Env: env}}
}

// VariantVal constructs a tagged-union value. It is the single constructor for
// enum, Result, and Option runtime values.
func VariantVal(typeID, tag string, fields map[string]Value) Value {
	if fields == nil {
		fields = map[string]Value{}
	}
	return Value{Kind: KindVariant, Variant: &Variant{TypeID: typeID, Tag: tag, Fields: fields}}
}

// Field reads a tagged-union payload field by name. ok is false when v is not a
// variant or the field is absent (a not-present read, never a panic).
func (v Value) Field(name string) (Value, bool) {
	if v.Kind != KindVariant || v.Variant == nil {
		return Value{}, false
	}
	f, ok := v.Variant.Fields[name]
	return f, ok
}

// Equal reports structural equality of two values. Primitives, structs, slices,
// maps, and variants compare by content; function values compare by identity
// (same *FuncValue pointer).
func (v Value) Equal(other Value) bool {
	if v.Kind != other.Kind {
		return false
	}
	switch v.Kind {
	case KindNil:
		return true
	case KindInt:
		return v.Int == other.Int
	case KindFloat:
		return v.Float == other.Float
	case KindString:
		return v.Str == other.Str
	case KindBool:
		return v.Bool == other.Bool
	case KindStruct:
		return structEqual(v.Struct, other.Struct)
	case KindSlice:
		if len(v.Slice) != len(other.Slice) {
			return false
		}
		for i := range v.Slice {
			if !v.Slice[i].Equal(other.Slice[i]) {
				return false
			}
		}
		return true
	case KindMap:
		return mapEqual(v.Map, other.Map)
	case KindFunc:
		return v.Func == other.Func
	case KindVariant:
		return variantEqual(v.Variant, other.Variant)
	default:
		return false
	}
}

func fieldsEqual(a, b map[string]Value) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || !av.Equal(bv) {
			return false
		}
	}
	return true
}

func structEqual(a, b *StructValue) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.TypeID == b.TypeID && fieldsEqual(a.Fields, b.Fields)
}

func mapEqual(a, b *MapValue) bool {
	if a == nil || b == nil {
		return a == b
	}
	return fieldsEqual(a.Entries, b.Entries)
}

func variantEqual(a, b *Variant) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.TypeID == b.TypeID && a.Tag == b.Tag && fieldsEqual(a.Fields, b.Fields)
}

// String renders a value to a readable, deterministic form.
func (v Value) String() string {
	switch v.Kind {
	case KindNil:
		return "nil"
	case KindInt:
		return strconv.FormatInt(v.Int, 10)
	case KindFloat:
		return strconv.FormatFloat(v.Float, 'g', -1, 64)
	case KindString:
		return strconv.Quote(v.Str)
	case KindBool:
		return strconv.FormatBool(v.Bool)
	case KindStruct:
		if v.Struct == nil {
			return "struct{}"
		}
		return v.Struct.TypeID + "{" + renderFields(v.Struct.Fields) + "}"
	case KindSlice:
		parts := make([]string, len(v.Slice))
		for i, e := range v.Slice {
			parts[i] = e.String()
		}
		return "[" + strings.Join(parts, " ") + "]"
	case KindMap:
		if v.Map == nil {
			return "map[]"
		}
		return "map[" + renderFields(v.Map.Entries) + "]"
	case KindFunc:
		name := ""
		if v.Func != nil {
			name = v.Func.Name
		}
		return "func " + name
	case KindVariant:
		if v.Variant == nil {
			return "<variant>"
		}
		s := v.Variant.TypeID + "." + v.Variant.Tag
		if len(v.Variant.Fields) > 0 {
			s += "(" + renderFields(v.Variant.Fields) + ")"
		}
		return s
	default:
		return "<" + v.Kind.String() + ">"
	}
}

// renderFields renders a name->Value map with keys sorted for determinism.
func renderFields(m map[string]Value) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + ": " + m[k].String()
	}
	return strings.Join(parts, ", ")
}
