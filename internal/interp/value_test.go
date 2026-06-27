package interp

import "testing"

// TestConstructEachKind constructs a Variant and every primitive/composite Value
// and asserts the resulting Kind is correct (AC: the Value type covers int,
// float, string, bool, nil, struct, slice, map, function, plus Variant).
func TestConstructEachKind(t *testing.T) {
	cases := []struct {
		name string
		v    Value
		want Kind
	}{
		{"int", IntVal(7), KindInt},
		{"float", FloatVal(3.5), KindFloat},
		{"string", StrVal("hi"), KindString},
		{"bool", BoolVal(true), KindBool},
		{"nil", NilVal(), KindNil},
		{"struct", StructVal("Point", map[string]Value{"X": IntVal(1)}), KindStruct},
		{"slice", SliceVal(IntVal(1), IntVal(2)), KindSlice},
		{"map", MapVal(map[string]Value{"a": IntVal(1)}), KindMap},
		{"func", FuncVal("main"), KindFunc},
		{"variant", VariantVal("Result", "Ok", map[string]Value{"value": IntVal(42)}), KindVariant},
	}
	for _, c := range cases {
		if c.v.Kind != c.want {
			t.Errorf("%s: Kind = %v, want %v", c.name, c.v.Kind, c.want)
		}
	}
}

// TestVariantUniformForSumTypes shows the ONE tagged union backs enum, Result,
// and Option uniformly (AC: Variant{TypeID,Tag,Fields} used uniformly).
func TestVariantUniformForSumTypes(t *testing.T) {
	result := VariantVal("Result", "Ok", map[string]Value{"value": IntVal(1)})
	option := VariantVal("Option", "Some", map[string]Value{"value": StrVal("x")})
	enum := VariantVal("Color", "Red", nil)

	for _, v := range []Value{result, option, enum} {
		if v.Kind != KindVariant || v.Variant == nil {
			t.Fatalf("expected a variant, got %v", v.Kind)
		}
	}
	if result.Variant.TypeID != "Result" || result.Variant.Tag != "Ok" {
		t.Errorf("result variant = %s.%s", result.Variant.TypeID, result.Variant.Tag)
	}
	if option.Variant.Tag != "Some" {
		t.Errorf("option tag = %s, want Some", option.Variant.Tag)
	}
	if enum.Variant.Tag != "Red" || len(enum.Variant.Fields) != 0 {
		t.Errorf("data-less enum variant mis-encoded: %s", enum)
	}
}

// TestFieldReadBack reads a Variant payload field by name (AC: reads a Variant
// field back by name), and confirms absent/non-variant reads report not-present.
func TestFieldReadBack(t *testing.T) {
	v := VariantVal("Result", "Ok", map[string]Value{"value": IntVal(42)})

	got, ok := v.Field("value")
	if !ok {
		t.Fatal("Field(value): ok = false, want true")
	}
	if !got.Equal(IntVal(42)) {
		t.Errorf("Field(value) = %s, want 42", got)
	}

	if _, ok := v.Field("missing"); ok {
		t.Error("Field(missing): ok = true, want false")
	}
	if _, ok := IntVal(1).Field("value"); ok {
		t.Error("Field on non-variant: ok = true, want false")
	}
}

// TestEqual asserts Value equality across kinds, including variants (AC: asserts
// Value equality).
func TestEqual(t *testing.T) {
	equalPairs := []struct {
		name string
		a, b Value
	}{
		{"int", IntVal(5), IntVal(5)},
		{"float", FloatVal(1.5), FloatVal(1.5)},
		{"string", StrVal("x"), StrVal("x")},
		{"bool", BoolVal(true), BoolVal(true)},
		{"nil", NilVal(), NilVal()},
		{"struct", StructVal("P", map[string]Value{"X": IntVal(1)}), StructVal("P", map[string]Value{"X": IntVal(1)})},
		{"slice", SliceVal(IntVal(1), IntVal(2)), SliceVal(IntVal(1), IntVal(2))},
		{"map", MapVal(map[string]Value{"a": IntVal(1)}), MapVal(map[string]Value{"a": IntVal(1)})},
		{"variant", VariantVal("Option", "Some", map[string]Value{"value": IntVal(9)}), VariantVal("Option", "Some", map[string]Value{"value": IntVal(9)})},
	}
	for _, c := range equalPairs {
		if !c.a.Equal(c.b) {
			t.Errorf("%s: Equal = false, want true (%s vs %s)", c.name, c.a, c.b)
		}
	}

	unequalPairs := []struct {
		name string
		a, b Value
	}{
		{"int-diff", IntVal(1), IntVal(2)},
		{"kind-diff", IntVal(1), StrVal("1")},
		{"variant-tag", VariantVal("Result", "Ok", nil), VariantVal("Result", "Err", nil)},
		{"variant-type", VariantVal("Result", "Ok", nil), VariantVal("Option", "Ok", nil)},
		{"variant-field", VariantVal("Result", "Ok", map[string]Value{"value": IntVal(1)}), VariantVal("Result", "Ok", map[string]Value{"value": IntVal(2)})},
		{"slice-len", SliceVal(IntVal(1)), SliceVal(IntVal(1), IntVal(2))},
		{"struct-field", StructVal("P", map[string]Value{"X": IntVal(1)}), StructVal("P", map[string]Value{"X": IntVal(9)})},
	}
	for _, c := range unequalPairs {
		if c.a.Equal(c.b) {
			t.Errorf("%s: Equal = true, want false (%s vs %s)", c.name, c.a, c.b)
		}
	}
}

// TestEqualFuncByIdentity confirms function values compare by identity.
func TestEqualFuncByIdentity(t *testing.T) {
	f := FuncVal("main")
	if !f.Equal(f) {
		t.Error("same func value should be equal to itself")
	}
	if f.Equal(FuncVal("main")) {
		t.Error("distinct func values with same name should not be equal (identity)")
	}
}

// TestString asserts every kind renders to a non-empty, readable string (AC:
// asserts String() rendering).
func TestString(t *testing.T) {
	cases := []struct {
		name string
		v    Value
		want string
	}{
		{"int", IntVal(42), "42"},
		{"float", FloatVal(3.5), "3.5"},
		{"string", StrVal("hi"), `"hi"`},
		{"bool", BoolVal(true), "true"},
		{"nil", NilVal(), "nil"},
		{"struct", StructVal("P", map[string]Value{"X": IntVal(1)}), "P{X: 1}"},
		{"slice", SliceVal(IntVal(1), IntVal(2)), "[1 2]"},
		{"map", MapVal(map[string]Value{"a": IntVal(1)}), "map[a: 1]"},
		{"variant-none", VariantVal("Option", "None", nil), "Option.None"},
		{"variant-payload", VariantVal("Result", "Ok", map[string]Value{"value": IntVal(7)}), "Result.Ok(value: 7)"},
	}
	for _, c := range cases {
		got := c.v.String()
		if got == "" {
			t.Errorf("%s: String() is empty", c.name)
		}
		if got != c.want {
			t.Errorf("%s: String() = %q, want %q", c.name, got, c.want)
		}
	}
	if got := FuncVal("main").String(); got == "" {
		t.Error("func: String() is empty")
	}
}
