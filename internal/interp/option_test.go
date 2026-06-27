package interp

// These tests prove US-016 "Eval Option as tagged union": the interpreter
// represents Option.Some / Option.None as the universal tagged-union Value —
// with no *T optimization — and matches over them, binding the unwrapped Some
// payload and binding nothing for None.
//
// Tests drive real parsed+resolved goal programs through newInterp (call_test.go)
// and the standard direct-evalExpr seam. stdlib testing only; no testify.

import (
	"strconv"
	"strings"
	"testing"

	"goal/internal/ast"
)

// 04-option shape: `first` returns Option.None for an empty slice and
// Option.Some(head) otherwise; `exists` matches over an Option result.
const optionProgram = `package num

func first(xs []int) Option[int] {
	if len(xs) == 0 {
		return Option.None
	}
	return Option.Some(xs[0])
}

func describe(xs []int) string {
	match first(xs) {
		Option.Some(v) => return "some"
		Option.None => return "none"
	}
}

func headOrZero(xs []int) int {
	match first(xs) {
		Option.Some(v) => return v
		Option.None => return 0
	}
}

func main() {}
`

// sliceLit builds a []int composite-literal argument from the given int literals.
func sliceLit(elems ...int) ast.Expr {
	lit := &ast.CompositeLit{Type: &ast.ArrayType{Elt: &ast.Ident{Name: "int"}}}
	for _, e := range elems {
		lit.Elts = append(lit.Elts, intLit(strconv.Itoa(e)))
	}
	return lit
}

// TestOptionSomeConstruction: Option.Some(x) evaluates to a tagged-union value
// tagged "Some" under TypeID "Option", carrying the present value as its single
// payload (no *T optimization).
func TestOptionSomeConstruction(t *testing.T) {
	ip := newInterp(t, optionProgram)
	got, err := ip.evalExpr(call("first", sliceLit(7, 8, 9)), ip.root)
	if err != nil {
		t.Fatalf("first([7,8,9]): %v", err)
	}
	if got.Kind != KindVariant || got.Variant == nil {
		t.Fatalf("first kind = %s, want variant", got.Kind)
	}
	if got.Variant.TypeID != "Option" || got.Variant.Tag != "Some" {
		t.Fatalf("first = %s.%s, want Option.Some", got.Variant.TypeID, got.Variant.Tag)
	}
	payload, ok := payloadValue(got.Variant)
	if !ok {
		t.Fatalf("Option.Some carries no single payload: %v", got)
	}
	if payload.Kind != KindInt || payload.Int != 7 {
		t.Fatalf("Some payload = %v, want int 7", payload)
	}
}

// TestOptionNoneConstruction: Option.None evaluates to a tagged-union value tagged
// "None" under TypeID "Option" carrying no payload.
func TestOptionNoneConstruction(t *testing.T) {
	ip := newInterp(t, optionProgram)
	got, err := ip.evalExpr(call("first", sliceLit()), ip.root)
	if err != nil {
		t.Fatalf("first([]): %v", err)
	}
	if got.Kind != KindVariant || got.Variant == nil {
		t.Fatalf("first kind = %s, want variant", got.Kind)
	}
	if got.Variant.TypeID != "Option" || got.Variant.Tag != "None" {
		t.Fatalf("first = %s.%s, want Option.None", got.Variant.TypeID, got.Variant.Tag)
	}
	if _, ok := payloadValue(got.Variant); ok {
		t.Fatalf("Option.None should carry no payload, got %v", got)
	}
}

// TestOptionMatchArms: a match over an Option runs the Some arm for a present
// value and the None arm for an absent value.
func TestOptionMatchArms(t *testing.T) {
	ip := newInterp(t, optionProgram)
	cases := []struct {
		name string
		arg  ast.Expr
		want string
	}{
		{"present", sliceLit(1), "some"},
		{"absent", sliceLit(), "none"},
	}
	for _, c := range cases {
		got, err := ip.evalExpr(call("describe", c.arg), ip.root)
		if err != nil {
			t.Fatalf("describe(%s): %v", c.name, err)
		}
		if got.Kind != KindString || got.Str != c.want {
			t.Fatalf("describe(%s) = %v, want %q", c.name, got, c.want)
		}
	}
}

// TestOptionSomeArmBindsUnwrappedValue: the Some arm binding is the UNWRAPPED
// inner value (read directly as `v`, not via `.field`), proving Option payload
// binding matches the Result/surface semantics.
func TestOptionSomeArmBindsUnwrappedValue(t *testing.T) {
	ip := newInterp(t, optionProgram)
	got, err := ip.evalExpr(call("headOrZero", sliceLit(42, 5)), ip.root)
	if err != nil {
		t.Fatalf("headOrZero([42,5]): %v", err)
	}
	if got.Kind != KindInt || got.Int != 42 {
		t.Fatalf("headOrZero([42,5]) = %v, want int 42", got)
	}
	zero, err := ip.evalExpr(call("headOrZero", sliceLit()), ip.root)
	if err != nil {
		t.Fatalf("headOrZero([]): %v", err)
	}
	if zero.Kind != KindInt || zero.Int != 0 {
		t.Fatalf("headOrZero([]) = %v, want int 0 (None arm)", zero)
	}
}

// TestOptionUnknownCtorIsRefused: an Option constructor other than Some/None is a
// located, descriptive error rather than a silent value.
func TestOptionUnknownCtorIsRefused(t *testing.T) {
	ip := newInterp(t, optionProgram)
	bad := &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: &ast.Ident{Name: "Option"}, Sel: &ast.Ident{Name: "Maybe"}},
		Args: []ast.Expr{intLit("1")},
	}
	_, err := ip.evalExpr(bad, ip.root)
	if err == nil {
		t.Fatalf("Option.Maybe(...) did not error")
	}
	if !strings.Contains(err.Error(), "unknown Option constructor") || !strings.Contains(err.Error(), "Maybe") {
		t.Fatalf("error = %q, want it to name the unknown constructor Maybe", err.Error())
	}
}

// TestOptionSomeArityIsRefused: Option.Some with other than one argument is a
// located, descriptive error.
func TestOptionSomeArityIsRefused(t *testing.T) {
	ip := newInterp(t, optionProgram)
	bad := &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: &ast.Ident{Name: "Option"}, Sel: &ast.Ident{Name: "Some"}},
		Args: nil, // zero args
	}
	_, err := ip.evalExpr(bad, ip.root)
	if err == nil {
		t.Fatalf("Option.Some() with no args did not error")
	}
	if !strings.Contains(err.Error(), "expects 1 argument") {
		t.Fatalf("error = %q, want an arity complaint", err.Error())
	}
}
