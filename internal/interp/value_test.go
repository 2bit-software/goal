package interp

import "testing"

// Tests for the Value payload accessors and predicates added in US-003. They
// pin that each accessor returns the CURRENT flat-struct payload and that the
// is-predicates agree with Kind (pointer-backed kinds also nil-check).

func TestValueAccessorsReturnPayload(t *testing.T) {
	if got := IntVal(42).asInt(); got != 42 {
		t.Errorf("asInt() = %d, want 42", got)
	}
	if got := FloatVal(3.5).asFloat(); got != 3.5 {
		t.Errorf("asFloat() = %v, want 3.5", got)
	}
	if got := StrVal("hi").asStr(); got != "hi" {
		t.Errorf("asStr() = %q, want %q", got, "hi")
	}
	if got := BoolVal(true).asBool(); got != true {
		t.Errorf("asBool() = %v, want true", got)
	}
	if got := NilVal().kind(); got != KindNil {
		t.Errorf("kind() = %v, want %v", got, KindNil)
	}
	if got := IntVal(1).kind(); got != KindInt {
		t.Errorf("kind() = %v, want %v", got, KindInt)
	}
}

func TestValueStructAccessor(t *testing.T) {
	sv := StructVal("T", map[string]Value{"x": IntVal(1)})
	s := sv.asStruct()
	if s == nil || s.TypeID != "T" {
		t.Fatalf("asStruct() = %+v, want TypeID T", s)
	}
	if v, ok := s.Fields["x"]; !ok || !v.Equal(IntVal(1)) {
		t.Errorf("asStruct().Fields[x] = %v, ok=%v", v, ok)
	}
	// Mutation through the accessor writes through to the same backing.
	s.Fields["x"] = IntVal(9)
	if v, _ := sv.asStruct().Fields["x"]; !v.Equal(IntVal(9)) {
		t.Errorf("mutation via asStruct not observed: got %v", v)
	}
	// Non-struct read returns nil.
	if IntVal(1).asStruct() != nil {
		t.Error("asStruct() on int should be nil")
	}
}

func TestValueSliceAccessor(t *testing.T) {
	sv := SliceVal(IntVal(1), IntVal(2))
	s := sv.asSlice()
	if len(s) != 2 {
		t.Fatalf("asSlice() len = %d, want 2", len(s))
	}
	// Mutation through the shared backing is observed.
	s[0] = IntVal(7)
	if !sv.asSlice()[0].Equal(IntVal(7)) {
		t.Errorf("mutation via asSlice not observed: got %v", sv.asSlice()[0])
	}
}

func TestValueMapAccessor(t *testing.T) {
	mv := MapVal(map[string]Value{"k": StrVal("v")})
	m := mv.asMap()
	if m == nil {
		t.Fatal("asMap() = nil")
	}
	if v, ok := m.Entries["k"]; !ok || !v.Equal(StrVal("v")) {
		t.Errorf("asMap().Entries[k] = %v, ok=%v", v, ok)
	}
	if IntVal(1).asMap() != nil {
		t.Error("asMap() on int should be nil")
	}
}

func TestValueFuncAccessor(t *testing.T) {
	fv := FuncVal("f")
	f := fv.asFunc()
	if f == nil || f.Name != "f" {
		t.Fatalf("asFunc() = %+v, want Name f", f)
	}
	if IntVal(1).asFunc() != nil {
		t.Error("asFunc() on int should be nil")
	}
}

func TestValueVariantAccessor(t *testing.T) {
	vv := VariantVal("Status", "Active", map[string]Value{"n": IntVal(1)})
	v := vv.asVariant()
	if v == nil || v.TypeID != "Status" || v.Tag != "Active" {
		t.Fatalf("asVariant() = %+v", v)
	}
	if IntVal(1).asVariant() != nil {
		t.Error("asVariant() on int should be nil")
	}
}

func TestValuePredicates(t *testing.T) {
	cases := []struct {
		name string
		v    Value
		pred func(Value) bool
		want bool
	}{
		{"isNil/nil", NilVal(), Value.isNil, true},
		{"isNil/int", IntVal(1), Value.isNil, false},
		{"isInt/int", IntVal(1), Value.isInt, true},
		{"isInt/str", StrVal("x"), Value.isInt, false},
		{"isFloat/float", FloatVal(1), Value.isFloat, true},
		{"isStr/str", StrVal("x"), Value.isStr, true},
		{"isBool/bool", BoolVal(true), Value.isBool, true},
		{"isStruct/struct", StructVal("T", nil), Value.isStruct, true},
		{"isStruct/int", IntVal(1), Value.isStruct, false},
		{"isSlice/slice", SliceVal(), Value.isSlice, true},
		{"isMap/map", MapVal(nil), Value.isMap, true},
		{"isMap/int", IntVal(1), Value.isMap, false},
		{"isFunc/func", FuncVal("f"), Value.isFunc, true},
		{"isVariant/variant", VariantVal("E", "A", nil), Value.isVariant, true},
		{"isVariant/int", IntVal(1), Value.isVariant, false},
	}
	for _, tc := range cases {
		if got := tc.pred(tc.v); got != tc.want {
			t.Errorf("%s = %v, want %v", tc.name, got, tc.want)
		}
	}
	// Pointer-backed predicates reject a mismatched-kind zero payload.
	if (Value{Kind: KindStruct}).isStruct() {
		t.Error("isStruct() should be false when Struct is nil")
	}
	if (Value{Kind: KindMap}).isMap() {
		t.Error("isMap() should be false when Map is nil")
	}
	if (Value{Kind: KindVariant}).isVariant() {
		t.Error("isVariant() should be false when Variant is nil")
	}
}
