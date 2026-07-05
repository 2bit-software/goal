package backendtest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// enumMethodFixture is the goal source that declares a sum type carrying
// value-receiver methods (US-001). It lives beside the test so the lowered-source
// assertions and the build-and-run parity check read the same program.
const enumMethodFixture = "../backend/testdata/enummethods/main.goal"

// enumMethodReference is a hand-authored Go program that models the same behaviour
// with an explicit interface + type switch. TestEnumMethodBehaviour asserts the
// transpiled program's stdout matches this reference's stdout.
const enumMethodReference = `package main

import "fmt"

type Shape interface{ isShape() }
type Circle struct{ r float64 }
type Rect struct {
	w, h float64
}

func (Circle) isShape() {}
func (Rect) isShape()   {}

func area(s Shape) float64 {
	switch v := s.(type) {
	case Circle:
		return 3 * v.r * v.r
	case Rect:
		return v.w * v.h
	}
	panic("unreachable")
}

func scaled(s Shape, factor float64) float64 { return area(s) * factor }

func dims(s Shape) (float64, bool) {
	var isRect bool
	switch v := s.(type) {
	case Circle:
		isRect = v.r < 0
	case Rect:
		isRect = v.w >= 0
	}
	return area(s), isRect
}

func main() {
	var c Shape = Circle{2}
	var r Shape = Rect{3, 4}
	cw, cRect := dims(c)
	rw, rRect := dims(r)
	fmt.Printf("%.1f %.1f %.1f %v\n", area(c), scaled(c, 2), cw, cRect)
	fmt.Printf("%.1f %.1f %.1f %v\n", area(r), scaled(r, 2), rw, rRect)
}
`

// TestEnumMethodLowering asserts the lowered Go for an enum with value-receiver
// methods: each method's signature is folded into the enum interface, each method
// body lives in exactly one shared free function, and each variant gets one
// forwarding method that delegates to that free function.
func TestEnumMethodLowering(t *testing.T) {
	goSrc := mustTranspile(t, readFixture(t, enumMethodFixture))

	// The interface method set folds in every method signature plus the marker.
	wantInterface := []string{"Area() float64", "Scaled(factor float64) float64", "Dims() (float64, bool)", "isShape()"}
	iface := interfaceBlock(goSrc, "Shape")
	if iface == "" {
		t.Fatalf("no `type Shape interface` block in lowered source:\n%s", goSrc)
	}
	for _, m := range wantInterface {
		if !strings.Contains(iface, m) {
			t.Errorf("enum interface is missing method %q; got:\n%s", m, iface)
		}
	}

	// Exactly one shared free function per method holds the body.
	for _, free := range []string{"func Shape_Area(s Shape) float64", "func Shape_Scaled(s Shape, factor float64) float64", "func Shape_Dims(s Shape) (float64, bool)"} {
		if got := strings.Count(goSrc, free); got != 1 {
			t.Errorf("expected exactly one shared free function %q, found %d:\n%s", free, got, goSrc)
		}
	}

	// One forwarding method per variant per method, delegating to the free function.
	forwarders := map[string]string{
		"func (s Shape_Circle) Area() float64":                 "return Shape_Area(s)",
		"func (s Shape_Rect) Area() float64":                   "return Shape_Area(s)",
		"func (s Shape_Circle) Scaled(factor float64) float64": "return Shape_Scaled(s, factor)",
		"func (s Shape_Rect) Scaled(factor float64) float64":   "return Shape_Scaled(s, factor)",
		"func (s Shape_Circle) Dims() (float64, bool)":         "return Shape_Dims(s)",
		"func (s Shape_Rect) Dims() (float64, bool)":           "return Shape_Dims(s)",
	}
	for sig, body := range forwarders {
		if got := strings.Count(goSrc, sig); got != 1 {
			t.Errorf("expected exactly one forwarding method %q, found %d:\n%s", sig, got, goSrc)
		}
		if !strings.Contains(goSrc, body) {
			t.Errorf("forwarding method body %q not found in lowered source:\n%s", body, goSrc)
		}
	}
}

// TestEnumMethodBehaviour builds and runs the transpiled fixture and asserts its
// stdout equals the hand-authored type-switch reference implementation.
func TestEnumMethodBehaviour(t *testing.T) {
	goSrc := mustTranspile(t, readFixture(t, enumMethodFixture))

	got := runGoMain(t, goSrc)
	want := runGoMain(t, enumMethodReference)
	if got != want {
		t.Errorf("transpiled output differs from reference type-switch:\ntranspiled:\n%s\nreference:\n%s", got, want)
	}
}

// readFixture returns the contents of a goal fixture file relative to the test.
func readFixture(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return string(b)
}

// interfaceBlock returns the `type <name> interface { ... }` block from goSrc, or
// "" when absent. It scans brace depth so a nested signature does not end it early.
func interfaceBlock(goSrc, name string) string {
	marker := "type " + name + " interface {"
	start := strings.Index(goSrc, marker)
	if start < 0 {
		return ""
	}
	depth := 0
	for i := start + len(marker) - 1; i < len(goSrc); i++ {
		switch goSrc[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return goSrc[start : i+1]
			}
		}
	}
	return goSrc[start:]
}

// runGoMain writes a `package main` Go source into an isolated module, runs it, and
// returns its stdout. It fails the test on any build or run error.
func runGoMain(t *testing.T, goSrc string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(goSrc), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module enummethodtest\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run failed: %v\noutput:\n%s\nsource:\n%s", err, out, goSrc)
	}
	return string(out)
}
