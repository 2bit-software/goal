package interp

// These tests prove US-018 "Eval implements dispatch": the goscript interpreter
// honors `implements`. Because the interpreter erases static types at runtime
// (REWRITE-ARCHITECTURE §3.2), an interface-typed binding simply holds the
// concrete value, and an interface method call dispatches structurally on the
// concrete value's type via the US-010 method registry (registerMethods /
// tryMethodCall / callMethod). These tests exercise that dispatch over a
// 07-implements shape: differently-typed value receivers through an interface
// parameter, a pointer receiver through an interface parameter (mutation
// observed), and a heterogeneous slice of interface values.
//
// Interface bodies use a single method or all-returning methods because the
// goal parser currently mis-parses a void interface method followed by another
// method (a pre-existing limitation, out of scope for this story).

import "testing"

// TestImplementsValueReceiverDifferentTypes: an interface method called on
// differently-typed values dispatches to each type's own value-receiver
// implementation (FR-1, FR-2, FR-3 value path).
func TestImplementsValueReceiverDifferentTypes(t *testing.T) {
	src := `package main

type Stringer interface {
	Describe() string
}

type Point struct implements Stringer {
	X int
	Y int
}

type Label struct implements Stringer {
	text string
}

func (p Point) Describe() string {
	return "point"
}

func (l Label) Describe() string {
	return "label"
}

func render(s Stringer) string {
	return s.Describe()
}

func viaPoint() string {
	return render(Point{X: 1, Y: 2})
}

func viaLabel() string {
	return render(Label{text: "hi"})
}
`
	ip := newInterp(t, src)
	if got := evalFn(t, ip, "viaPoint"); got.Kind != KindString || got.Str != "point" {
		t.Fatalf("viaPoint = %#v, want string \"point\"", got)
	}
	if got := evalFn(t, ip, "viaLabel"); got.Kind != KindString || got.Str != "label" {
		t.Fatalf("viaLabel = %#v, want string \"label\"", got)
	}
}

// TestImplementsPointerReceiverThroughInterface: a pointer-receiver method
// dispatched through an interface parameter observes and mutates the underlying
// value (FR-3 pointer path). No `&` is needed — goal struct values share their
// underlying struct value.
func TestImplementsPointerReceiverThroughInterface(t *testing.T) {
	src := `package main

type Resetter interface {
	Reset()
}

type Counter struct implements Resetter {
	n int
}

func (c *Counter) Reset() {
	c.n = 0
}

func use(r Resetter) {
	r.Reset()
}

func driver() int {
	c := Counter{n: 5}
	use(c)
	return c.n
}
`
	ip := newInterp(t, src)
	if got := evalFn(t, ip, "driver"); got.Kind != KindInt || got.Int != 0 {
		t.Fatalf("driver = %#v, want int 0 (pointer-receiver Reset ran through the interface)", got)
	}
}

// TestImplementsHeterogeneousSliceDispatch: a method called on each element of a
// heterogeneous slice of interface values dispatches to that element's concrete
// implementation (FR-4).
func TestImplementsHeterogeneousSliceDispatch(t *testing.T) {
	src := `package main

type Shape interface {
	Area() int
}

type Square struct implements Shape {
	side int
}

type Rect struct implements Shape {
	w int
	h int
}

func (s Square) Area() int {
	return s.side * s.side
}

func (r Rect) Area() int {
	return r.w * r.h
}

func total() int {
	shapes := []Shape{Square{side: 3}, Rect{w: 2, h: 4}}
	sum := 0
	for _, s := range shapes {
		sum = sum + s.Area()
	}
	return sum
}
`
	ip := newInterp(t, src)
	// 3*3 (Square) + 2*4 (Rect) = 17 — proves each concrete Area ran.
	if got := evalFn(t, ip, "total"); got.Kind != KindInt || got.Int != 17 {
		t.Fatalf("total = %#v, want int 17", got)
	}
}

// TestImplementsMissingMethodIsLoud: calling a method the concrete value's type
// does not declare is a descriptive interpreter error, not a silent nil. Such
// programs are rejected statically by sema; the runtime stays loud.
func TestImplementsMissingMethodIsLoud(t *testing.T) {
	src := `package main

type Stringer interface {
	Describe() string
}

type Plain struct {
	x int
}

func render(s Stringer) string {
	return s.Describe()
}

func boom() string {
	return render(Plain{x: 1})
}
`
	ip := newInterp(t, src)
	// Plain has no Describe method, so dispatch must fail loudly rather than
	// returning a silent zero value. Build the call directly so the helper's
	// t.Fatalf-on-error does not mask the expected failure.
	if _, err := ip.evalExpr(call("boom"), ip.root); err == nil {
		t.Fatalf("boom: expected a loud error for the missing Describe method, got none")
	}
}
