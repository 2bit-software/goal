package interp

// These tests prove US-020 "Eval defaults expansion": the interpreter expands a
// `...defaults` spread in a struct composite literal at construction, filling
// every field the literal did not set explicitly with that field's safe runtime
// zero value while preserving the explicitly set fields. They drive real
// parsed+resolved goal programs (modeled on features/08-no-zero-value) through
// the direct-evalExpr testing seam (newInterp + evalFn, from call_test.go /
// composite_test.go). A non-`defaults` spread is a descriptive refusal. stdlib
// testing only (no testify).

import (
	"strings"
	"testing"

	"goal/internal/ast"
)

// structField reads a named field off a returned struct value, failing the test
// when the value is not a struct or the field is absent.
func structField(t *testing.T, v Value, name string) Value {
	t.Helper()
	if v.Kind != KindStruct || v.Struct == nil {
		t.Fatalf("expected a struct value, got %s (%s)", v.Kind, v.String())
	}
	f, ok := v.Struct.Fields[name]
	if !ok {
		t.Fatalf("struct %s has no field %s", v.Struct.TypeID, name)
	}
	return f
}

// defaultsPrimitivesProgram mirrors features/08-no-zero-value defaults_primitives:
// newMember sets the no-default fields (name, role) explicitly and opts the rest
// into their zero values with `...defaults`.
const defaultsPrimitivesProgram = `package users

type User struct {
	name   string
	email  string
	active bool
	logins int
}

func newMember() User {
	return User{name: "ada", logins: 3, ...defaults}
}

func main() {}
`

func TestDefaultsFillsOmittedPrimitivesAndPreservesSet(t *testing.T) {
	ip := newInterp(t, defaultsPrimitivesProgram)
	got := evalFn(t, ip, "newMember")

	// Explicitly set fields are preserved.
	if name := structField(t, got, "name"); name.Kind != KindString || name.Str != "ada" {
		t.Errorf("name: want \"ada\", got %s", name.String())
	}
	if logins := structField(t, got, "logins"); logins.Kind != KindInt || logins.Int != 3 {
		t.Errorf("logins (explicit): want 3, got %s", logins.String())
	}
	// Omitted fields receive their safe zeros.
	if email := structField(t, got, "email"); email.Kind != KindString || email.Str != "" {
		t.Errorf("email default: want empty string, got %s", email.String())
	}
	if active := structField(t, got, "active"); active.Kind != KindBool || active.Bool {
		t.Errorf("active default: want false, got %s", active.String())
	}
}

// defaultsOrderingProgram puts `...defaults` BEFORE an explicit field to prove
// the fill is position-independent: an explicit field is never overwritten by a
// default, regardless of where the spread sits in the literal.
const defaultsOrderingProgram = `package users

type User struct {
	name   string
	email  string
	logins int
}

func mixed() User {
	return User{...defaults, email: "root@x"}
}

func main() {}
`

func TestDefaultsDoesNotOverwriteExplicitFieldBeforeOrAfter(t *testing.T) {
	ip := newInterp(t, defaultsOrderingProgram)
	got := evalFn(t, ip, "mixed")

	if email := structField(t, got, "email"); email.Kind != KindString || email.Str != "root@x" {
		t.Errorf("email (explicit, after spread): want \"root@x\", got %s", email.String())
	}
	if name := structField(t, got, "name"); name.Kind != KindString || name.Str != "" {
		t.Errorf("name default: want empty string, got %s", name.String())
	}
	if logins := structField(t, got, "logins"); logins.Kind != KindInt || logins.Int != 0 {
		t.Errorf("logins default: want 0, got %s", logins.String())
	}
}

// defaultsRefsProgram mirrors features/08-no-zero-value defaults_refs: a named
// struct field and a slice field are recovered by `...defaults` (Addr{} and an
// empty slice), while the unsafe-zero fields are set explicitly.
const defaultsRefsProgram = `package config

type Addr struct {
	host string
	port int
}

type Settings struct {
	primary  Addr
	meta     Addr
	tags     []string
	retries  int
}

func defaultSettings() Settings {
	return Settings{primary: Addr{host: "localhost", port: 8080}, ...defaults}
}

func main() {}
`

func TestDefaultsFillsNestedStructAndSlice(t *testing.T) {
	ip := newInterp(t, defaultsRefsProgram)
	got := evalFn(t, ip, "defaultSettings")

	// Explicit named-struct field preserved.
	primary := structField(t, got, "primary")
	if host := structField(t, primary, "host"); host.Kind != KindString || host.Str != "localhost" {
		t.Errorf("primary.host: want \"localhost\", got %s", host.String())
	}
	if port := structField(t, primary, "port"); port.Kind != KindInt || port.Int != 8080 {
		t.Errorf("primary.port: want 8080, got %s", port.String())
	}

	// Omitted named-struct field -> recursively zero-filled Addr{host:"", port:0}.
	meta := structField(t, got, "meta")
	if meta.Kind != KindStruct || meta.Struct == nil || meta.Struct.TypeID != "Addr" {
		t.Fatalf("meta default: want Addr struct, got %s", meta.String())
	}
	if host := structField(t, meta, "host"); host.Kind != KindString || host.Str != "" {
		t.Errorf("meta.host default: want empty string, got %s", host.String())
	}
	if port := structField(t, meta, "port"); port.Kind != KindInt || port.Int != 0 {
		t.Errorf("meta.port default: want 0, got %s", port.String())
	}

	// Omitted slice field -> empty slice (usable nil-slice equivalent).
	tags := structField(t, got, "tags")
	if tags.Kind != KindSlice {
		t.Fatalf("tags default: want slice, got %s", tags.String())
	}
	if len(tags.Slice) != 0 {
		t.Errorf("tags default: want empty slice, got %s", tags.String())
	}
	if retries := structField(t, got, "retries"); retries.Kind != KindInt || retries.Int != 0 {
		t.Errorf("retries default: want 0, got %s", retries.String())
	}
}

// TestNonDefaultsSpreadIsRefused proves a spread element other than
// `...defaults` (here `...derive`) is a loud, located refusal rather than a
// silent skip or a wrong value. The literal is hand-built since `...derive`
// would need a derive context to appear in source.
func TestNonDefaultsSpreadIsRefused(t *testing.T) {
	ip := newInterp(t, defaultsPrimitivesProgram) // any program with a known User type
	lit := &ast.CompositeLit{
		Type: &ast.Ident{Name: "User"},
		Elts: []ast.Expr{
			&ast.SpreadElement{X: &ast.Ident{Name: "derive"}},
		},
	}
	_, err := ip.evalExpr(lit, ip.root)
	if err == nil {
		t.Fatalf("expected a refusal for a ...derive spread, got none")
	}
	if !strings.Contains(err.Error(), "spread") || !strings.Contains(err.Error(), "defaults") {
		t.Errorf("error should name the unsupported spread and ...defaults, got: %v", err)
	}
}

// TestDefaultsUnknownStructIsRefused proves a `...defaults` on a struct type the
// front-end cannot resolve is a descriptive refusal, never a silent empty fill.
func TestDefaultsUnknownStructIsRefused(t *testing.T) {
	ip := newInterp(t, defaultsPrimitivesProgram)
	lit := &ast.CompositeLit{
		Type: &ast.Ident{Name: "Unknown"},
		Elts: []ast.Expr{
			&ast.SpreadElement{X: &ast.Ident{Name: "defaults"}},
		},
	}
	_, err := ip.evalExpr(lit, ip.root)
	if err == nil {
		t.Fatalf("expected a refusal for ...defaults on an unknown struct, got none")
	}
	if !strings.Contains(err.Error(), "unknown struct type") {
		t.Errorf("error should name the unknown struct type, got: %v", err)
	}
}
