package interp

// These tests prove US-021 "Eval derive and from": the interpreter evaluates
// `derive func` conversions field-by-field over the resolved sema facts —
// copying identity fields through, bridging differing fields through a registered
// `from func` (the from-registry), and recursing into nested in-file structs —
// and threads a fallible conversion's error through a `(T, error)` derive. An
// unsourced target field is a loud refusal, never a silent zero. They drive real
// parsed+resolved goal programs (modeled on features/12-derive-convert) through
// the direct-eval testing seam (newInterp + call, from call_test.go). stdlib
// testing only (no testify).

import (
	"strings"
	"testing"
)

// deriveNestedProgram mirrors features/12-derive-convert/derive_nested_struct: a
// total `upgrade` derive that copies Name (identity), recurses Home (Addr ->
// AddrV2), and bridges Zip (string -> Code) through the `from func parseCode`.
const deriveNestedProgram = `package app

type Code struct {
	v string
}

from func parseCode(s string) Code {
	return Code{v: s}
}

type Addr struct {
	Street string
	Zip    string
}

type AddrV2 struct {
	Street string
	Zip    Code
}

type Person struct {
	Name string
	Home Addr
}

type PersonV2 struct {
	Name string
	Home AddrV2
}

derive func upgrade(p Person) PersonV2

func makeV2() PersonV2 {
	return upgrade(Person{Name: "Ada", Home: Addr{Street: "Main", Zip: "90210"}})
}

func main() {}
`

func TestDeriveTotalNestedStructAndRegistryBridge(t *testing.T) {
	ip := newInterp(t, deriveNestedProgram)
	got := evalFn(t, ip, "makeV2")

	if got.Kind != KindStruct || got.Struct == nil || got.Struct.TypeID != "PersonV2" {
		t.Fatalf("want a PersonV2 struct, got %s (%s)", got.Kind, got.String())
	}
	// Identity field copied through.
	if name := structField(t, got, "Name"); name.Kind != KindString || name.Str != "Ada" {
		t.Errorf("Name (identity): want \"Ada\", got %s", name.String())
	}
	// Nested struct recursion: Home is an AddrV2.
	home := structField(t, got, "Home")
	if home.Kind != KindStruct || home.Struct == nil || home.Struct.TypeID != "AddrV2" {
		t.Fatalf("Home: want an AddrV2 struct, got %s (%s)", home.Kind, home.String())
	}
	if street := structField(t, home, "Street"); street.Kind != KindString || street.Str != "Main" {
		t.Errorf("Home.Street (identity): want \"Main\", got %s", street.String())
	}
	// Registry bridge through nested recursion: Zip (string) -> Code{v: "90210"}.
	zip := structField(t, home, "Zip")
	if zip.Kind != KindStruct || zip.Struct == nil || zip.Struct.TypeID != "Code" {
		t.Fatalf("Home.Zip: want a Code struct, got %s (%s)", zip.Kind, zip.String())
	}
	if v := structField(t, zip, "v"); v.Kind != KindString || v.Str != "90210" {
		t.Errorf("Home.Zip.v (bridged): want \"90210\", got %s", v.String())
	}
}

// deriveFallibleProgram exercises a fallible derive: Key bridges string -> ID
// through `from func parseID` which returns `(ID, error)`. A non-empty key
// succeeds; an empty key returns the conversion's error.
const deriveFallibleProgram = `package app

import "errors"

type ID struct {
	v string
}

from func parseID(s string) (ID, error) {
	if s == "" {
		return ID{}, errors.New("empty id")
	}
	return ID{v: s}, nil
}

type Raw struct {
	Name string
	Key  string
}

type Typed struct {
	Name string
	Key  ID
}

derive func toTyped(r Raw) (Typed, error)

func goodCase() (Typed, error) {
	return toTyped(Raw{Name: "n", Key: "k"})
}

func badCase() (Typed, error) {
	return toTyped(Raw{Name: "n", Key: ""})
}

func main() {}
`

func TestDeriveFallibleSucceedsAndReturnsNilError(t *testing.T) {
	ip := newInterp(t, deriveFallibleProgram)
	vals, err := ip.evalCallMulti(call("goodCase"), ip.root)
	if err != nil {
		t.Fatalf("goodCase: unexpected error: %v", err)
	}
	if len(vals) != 2 {
		t.Fatalf("goodCase: want 2 results (Typed, error), got %d", len(vals))
	}
	typed := vals[0]
	if typed.Kind != KindStruct || typed.Struct == nil || typed.Struct.TypeID != "Typed" {
		t.Fatalf("goodCase result: want a Typed struct, got %s (%s)", typed.Kind, typed.String())
	}
	if name := structField(t, typed, "Name"); name.Kind != KindString || name.Str != "n" {
		t.Errorf("Typed.Name (identity): want \"n\", got %s", name.String())
	}
	key := structField(t, typed, "Key")
	if key.Kind != KindStruct || key.Struct == nil || key.Struct.TypeID != "ID" {
		t.Fatalf("Typed.Key: want an ID struct, got %s (%s)", key.Kind, key.String())
	}
	if v := structField(t, key, "v"); v.Kind != KindString || v.Str != "k" {
		t.Errorf("Typed.Key.v (fallible bridge): want \"k\", got %s", v.String())
	}
	if vals[1].Kind != KindNil {
		t.Errorf("goodCase error result: want nil, got %s", vals[1].String())
	}
}

func TestDeriveFalliblePropagatesConversionError(t *testing.T) {
	ip := newInterp(t, deriveFallibleProgram)
	vals, err := ip.evalCallMulti(call("badCase"), ip.root)
	if err != nil {
		t.Fatalf("badCase: unexpected interpreter error: %v", err)
	}
	if len(vals) != 2 {
		t.Fatalf("badCase: want 2 results (Typed, error), got %d", len(vals))
	}
	// The fallible conversion error is propagated as the derive's second result.
	if vals[1].Kind != KindStruct || vals[1].Struct == nil {
		t.Fatalf("badCase error result: want a non-nil error value, got %s (%s)", vals[1].Kind, vals[1].String())
	}
	if msg := structField(t, vals[1], "message"); msg.Kind != KindString || msg.Str != "empty id" {
		t.Errorf("badCase error message: want \"empty id\", got %s", msg.String())
	}
}

// deriveUnsourcedProgram has a target field (B) with no same-named source field,
// so the derivation must refuse loudly rather than silently zero it.
const deriveUnsourcedProgram = `package app

type Src struct {
	A string
}

type Dst struct {
	A string
	B string
}

derive func widen(s Src) Dst

func runBad() Dst {
	return widen(Src{A: "x"})
}

func main() {}
`

func TestDeriveUnsourcedFieldIsRefused(t *testing.T) {
	ip := newInterp(t, deriveUnsourcedProgram)
	err := evalFnErr(t, ip, "runBad")
	if !strings.Contains(err.Error(), "not sourced") {
		t.Errorf("want a 'not sourced' refusal, got %v", err)
	}
	if !strings.Contains(err.Error(), "B") {
		t.Errorf("want the refusal to name the unsourced field B, got %v", err)
	}
}
